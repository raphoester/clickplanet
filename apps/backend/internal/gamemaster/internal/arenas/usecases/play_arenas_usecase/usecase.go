package play_arenas_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Planner interface {
	Plan(ctx context.Context) (moments.Plan, error)
}

type Judged interface {
	KeepJudged(ctx context.Context, judgement moments.Judgement) (bool, error)
}

// Planet says whether a rule set makes an arena battle.
type Planet interface {
	Arena(ctx context.Context, set schedule.SetName) (bool, error)
}

type Commands interface {
	Send(ctx context.Context, command moments.ArenaCommand) error
}

// Tick is what one Execute did: the battle it judged, and each arena command it sent.
type Tick struct {
	Judged []moments.Judgement
	Sent   []moments.ArenaCommand
}

type Executor interface {
	Execute(ctx context.Context) (Tick, error)
}

func New(planner Planner, judged Judged, planet Planet, commands Commands, clock cptime.Clock) *UseCase {
	return &UseCase{
		planner: planner, judged: judged, planet: planet, commands: commands, clock: clock,
		sent: cpcolls.NewSet[moments.ArenaKey](),
	}
}

// UseCase judges the battle that runs, once, and sends each arena command whose time has come, once per process and in
// order: a command Redis refused stops the tick, and it and every one after it go at the next Execute. Planet keeps
// each arena, so a boot that sends them again changes nothing.
type UseCase struct {
	planner  Planner
	judged   Judged
	planet   Planet
	commands Commands
	clock    cptime.Clock

	sent *cpcolls.Set[moments.ArenaKey]
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context) (Tick, error) {
	var tick Tick
	plan, err := u.planner.Plan(ctx)
	if err != nil {
		return tick, err //nolint:wrapcheck // the planner names it.
	}
	now := u.clock.Now()

	if battle, ok := plan.Unjudged(now); ok {
		inArena, err := u.planet.Arena(ctx, battle.Set)
		if err != nil {
			return tick, err //nolint:wrapcheck // the adapter names the set.
		}
		judgement := plan.Judge(battle, inArena)
		if _, err := u.judged.KeepJudged(ctx, judgement); err != nil {
			return tick, fmt.Errorf("failed to keep %s judged: %w", battle, err)
		}
		tick.Judged = append(tick.Judged, judgement)
		if plan, err = u.planner.Plan(ctx); err != nil {
			return tick, err //nolint:wrapcheck // the planner names it.
		}
	}

	for _, command := range plan.ArenasDueAt(now) {
		if u.sent.Contains(command.Key()) {
			continue
		}
		if err := u.commands.Send(ctx, command); err != nil {
			return tick, err //nolint:wrapcheck // the adapter names the arena.
		}
		u.sent.Add(command.Key())
		tick.Sent = append(tick.Sent, command)
	}
	return tick, nil
}

package play_resets_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Planner interface {
	Plan(ctx context.Context) (moments.Plan, error)
}

type Commands interface {
	// Reset asks planet to start the season on a blank map; planet wipes it once per season.
	Reset(ctx context.Context, starts season.Window) error
}

// Execute answers the season whose start it told, and told is true when it told one.
type Executor interface {
	Execute(ctx context.Context) (starts season.Window, told bool, err error)
}

func New(planner Planner, commands Commands, clock cptime.Clock) *UseCase {
	return &UseCase{planner: planner, commands: commands, clock: clock, told: cpcolls.NewSet[season.Number]()}
}

// UseCase tells planet once per process that the running season starts on a blank map. Planet keeps each reset, so
// a boot that tells it again wipes nothing; a command Redis did not take is sent again at the next Execute. A season
// that starts after a siege waits for the map from before the siege to be back, or the siege's end would undo the
// reset.
type UseCase struct {
	planner  Planner
	commands Commands
	clock    cptime.Clock

	told *cpcolls.Set[season.Number]
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context) (season.Window, bool, error) {
	plan, err := u.planner.Plan(ctx)
	if err != nil {
		return season.Window{}, false, err //nolint:wrapcheck // the planner names it.
	}
	book, ok := plan.BookAt(u.clock.Now())
	running := book.Window
	if !ok || !running.StartsOnBlankMap() || u.told.Contains(running.Number) || plan.Waits(running.StartsAt) {
		return season.Window{}, false, nil
	}
	if err := u.commands.Reset(ctx, running); err != nil {
		return running, false, err //nolint:wrapcheck // the adapter names the season.
	}
	u.told.Add(running.Number)
	return running, true, nil
}

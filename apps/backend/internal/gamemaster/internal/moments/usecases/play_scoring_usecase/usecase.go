package play_scoring_usecase

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Opened interface {
	KeepOpened(ctx context.Context, round moments.RoundID) error
}

type Commands interface {
	Send(ctx context.Context, command moments.Command) error
}

// Execute answers the commands it sent this time.
type Executor interface {
	Execute(ctx context.Context, plan moments.Plan) (sent []moments.Command, err error)
}

func New(opened Opened, commands Commands, clock cptime.Clock, again time.Duration) *UseCase {
	return &UseCase{opened: opened, commands: commands, clock: clock, again: again, sent: map[moments.Key]time.Time{}}
}

// UseCase sends scoring each command the plan says is due: an open once per process, and a close again each `again`
// until scoring answers it, scored, closed or missed, and the plan no longer says it is due. A command Redis did not
// take, and every one after it, is sent at the next Execute, so scoring hears them in order. A round is kept opened
// before it is opened.
type UseCase struct {
	opened   Opened
	commands Commands
	clock    cptime.Clock
	again    time.Duration

	sent map[moments.Key]time.Time
}

func (u *UseCase) Execute(ctx context.Context, plan moments.Plan) ([]moments.Command, error) {
	now := u.clock.Now()
	var sent []moments.Command
	for _, command := range plan.DueAt(now) {
		if last, ok := u.sent[command.Key()]; ok && (!command.Action.Closes() || now.Sub(last) < u.again) {
			continue
		}
		if command.Action == moments.OpenRound {
			if err := u.opened.KeepOpened(ctx, command.Round); err != nil {
				return sent, err //nolint:wrapcheck // the store names the round.
			}
		}
		if err := u.commands.Send(ctx, command); err != nil {
			return sent, err //nolint:wrapcheck // the adapter names the command.
		}
		u.sent[command.Key()] = now
		sent = append(sent, command)
	}
	return sent, nil
}

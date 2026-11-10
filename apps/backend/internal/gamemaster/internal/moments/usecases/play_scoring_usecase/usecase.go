package play_scoring_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
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

func New(opened Opened, commands Commands, clock cptime.Clock) *UseCase {
	return &UseCase{opened: opened, commands: commands, clock: clock, sent: cpcolls.NewSet[moments.Key]()}
}

// UseCase sends scoring each command once its time has come, once per process. A command Redis did not take, and every
// one after it, is sent at the next Execute, so scoring hears them in order. A round is kept opened before it is opened.
type UseCase struct {
	opened   Opened
	commands Commands
	clock    cptime.Clock

	sent *cpcolls.Set[moments.Key]
}

func (u *UseCase) Execute(ctx context.Context, plan moments.Plan) ([]moments.Command, error) {
	now := u.clock.Now()
	var sent []moments.Command
	for _, command := range plan.DueAt(now) {
		if u.sent.Contains(command.Key()) {
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
		u.sent.Add(command.Key())
		sent = append(sent, command)
	}
	return sent, nil
}

package tell_starts_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Told interface {
	KeepToldSeason(ctx context.Context, book moments.BookID) (bool, error)
	KeepToldDay(ctx context.Context, day moments.ToldDay) (bool, error)
	KeepToldBattle(ctx context.Context, round moments.RoundID) (bool, error)
}

// Execute answers what it told this time: the season, the day, then the battle.
type Executor interface {
	Execute(ctx context.Context, plan moments.Plan) ([]moments.Moment, error)
}

func New(told Told, events moments.Events, transactioner cpatomic.Transactioner, clock cptime.Clock) *UseCase {
	return &UseCase{told: told, events: events, transactioner: transactioner, clock: clock}
}

// UseCase tells the season, the day and the battle that run, each once as it starts, and a day again when its end
// moved. Level-triggered: a boot tells what runs if it was never told, and never what ended.
type UseCase struct {
	told          Told
	events        moments.Events
	transactioner cpatomic.Transactioner
	clock         cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, plan moments.Plan) ([]moments.Moment, error) {
	now := u.clock.Now()

	var told []moments.Moment
	err := u.transactioner.InTx(ctx, func(ctx context.Context) error {
		if book, ok := plan.BookAt(now); ok {
			kept, err := u.told.KeepToldSeason(ctx, book.ID())
			if err != nil {
				return err //nolint:wrapcheck // the store names the season.
			}
			if kept {
				told = append(told, moments.SeasonStarted{Window: book.Window})
			}
		}
		if day, ok := plan.DayAt(now); ok {
			kept, err := u.told.KeepToldDay(ctx, moments.ToldDay{Round: moments.DayRoundOf(day), EndsAt: day.EndsAt})
			if err != nil {
				return err //nolint:wrapcheck // the store names the day.
			}
			if kept {
				told = append(told, moments.DayStarted{Day: day})
			}
		}
		if battle, ok := plan.BattleAt(now); ok {
			kept, err := u.told.KeepToldBattle(ctx, battle.Round())
			if err != nil {
				return err //nolint:wrapcheck // the store names the battle.
			}
			if kept {
				told = append(told, moments.BattleStarted{Battle: battle})
			}
		}
		for _, moment := range told {
			if err := u.events.Record(ctx, moment); err != nil {
				return err //nolint:wrapcheck // the adapter names the event.
			}
		}
		return nil
	})
	if err != nil {
		return nil, err //nolint:wrapcheck // the transaction's own error.
	}
	return told, nil
}

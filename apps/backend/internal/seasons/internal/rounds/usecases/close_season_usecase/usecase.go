package close_season_usecase

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/timestamppb"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Seasons interface {
	ClosedIn(ctx context.Context, season calendar.Number) ([]rounds.Closed, error)
	End(ctx context.Context, season calendar.Season) (bool, error)
}

type UseCase struct {
	transactioner cpatomic.Transactioner
	store         Seasons
	outbox        cpatomic.Outbox
}

func New(transactioner cpatomic.Transactioner, store Seasons, outbox cpatomic.Outbox) *UseCase {
	return &UseCase{transactioner: transactioner, store: store, outbox: outbox}
}

// Execute ends a season from every round it closed, and tells its table once: ended is false when it ended before.
// Everything is read before the season is ended, so a failed read leaves it to end at the next try.
func (u *UseCase) Execute(ctx context.Context, season calendar.Season) (rounds.SeasonEnd, bool, error) {
	closed, err := u.store.ClosedIn(ctx, season.Number)
	if err != nil {
		return rounds.SeasonEnd{}, false, fmt.Errorf("failed to read the rounds the season closed: %w", err)
	}

	end := rounds.SeasonEndOf(season, closed)
	ended := false
	err = u.transactioner.InTx(ctx, func(ctx context.Context) error {
		var err error
		if ended, err = u.store.End(ctx, season); err != nil || !ended {
			return err //nolint:wrapcheck // the store says what failed.
		}
		if err := u.outbox.Record(ctx, eventOf(end)); err != nil {
			return fmt.Errorf("failed to record the season's end: %w", err)
		}
		return nil
	})
	if err != nil {
		return rounds.SeasonEnd{}, false, err
	}
	return end, ended, nil
}

func eventOf(end rounds.SeasonEnd) *seasonsv1.SeasonClosed {
	table := make([]*seasonsv1.SeasonPlace, 0, len(end.Table))
	for _, placed := range end.Table {
		table = append(table, &seasonsv1.SeasonPlace{
			Country:      string(placed.Country),
			Rank:         placed.Rank,
			Points:       uint32(placed.Points), //nolint:gosec // 25 a day for a few weeks.
			RoundsWon:    placed.RoundsWon,
			FinalePoints: uint32(placed.FinalePoints), //nolint:gosec // 250 a finale at most.
		})
	}
	return &seasonsv1.SeasonClosed{
		Season:  uint32(end.Season),
		EndedAt: timestamppb.New(end.EndedAt),
		Table:   table,
	}
}

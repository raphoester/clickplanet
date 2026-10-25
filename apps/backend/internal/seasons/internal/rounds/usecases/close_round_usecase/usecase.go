package close_round_usecase

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Rounds interface {
	Held(ctx context.Context, round rounds.Round) (map[rounds.Country]uint64, error)
	Number(ctx context.Context, round rounds.Round) (uint32, error)
	ClosedIn(ctx context.Context, season calendar.Number) ([]rounds.Closed, error)
	Close(ctx context.Context, round rounds.Round, results []rounds.Result) (bool, error)
}

type UseCase struct {
	transactioner cpatomic.Transactioner
	store         Rounds
	outbox        cpatomic.Outbox
}

func New(transactioner cpatomic.Transactioner, store Rounds, outbox cpatomic.Outbox) *UseCase {
	return &UseCase{transactioner: transactioner, store: store, outbox: outbox}
}

func (u *UseCase) Execute(ctx context.Context, round rounds.Round) (rounds.Closed, error) {
	held, err := u.store.Held(ctx, round)
	if err != nil {
		return rounds.Closed{}, fmt.Errorf("failed to read what each country held in the round: %w", err)
	}
	number, err := u.store.Number(ctx, round)
	if err != nil {
		return rounds.Closed{}, fmt.Errorf("failed to number the round: %w", err)
	}
	earlier, err := u.store.ClosedIn(ctx, round.Season)
	if err != nil {
		return rounds.Closed{}, fmt.Errorf("failed to read the rounds the season closed: %w", err)
	}

	closing := round.Closed(number, held)
	told := []proto.Message{eventOf(closing)}
	if end, over := closing.SeasonEnd(earlier); over {
		told = append(told, seasonEventOf(end))
	}

	err = u.transactioner.InTx(ctx, func(ctx context.Context) error {
		closed, err := u.store.Close(ctx, round, closing.Results)
		if err != nil {
			return fmt.Errorf("failed to close the round: %w", err)
		}
		if !closed {
			return nil
		}
		if err := u.outbox.Record(ctx, told...); err != nil {
			return fmt.Errorf("failed to record the round's close: %w", err)
		}
		return nil
	})
	if err != nil {
		return rounds.Closed{}, err
	}
	return closing, nil
}

func eventOf(closed rounds.Closed) *seasonsv1.RoundClosed {
	results := make([]*seasonsv1.RoundResult, 0, len(closed.Results))
	for _, result := range closed.Results {
		results = append(results, &seasonsv1.RoundResult{
			Country: string(result.Country),
			Rank:    result.Rank,
			Points:  result.Points,
		})
	}
	return &seasonsv1.RoundClosed{
		Season:  uint32(closed.Round.Season),
		Number:  closed.Number,
		Finale:  closed.Round.Finale,
		EndedAt: timestamppb.New(closed.Round.EndsAt),
		Results: results,
	}
}

func seasonEventOf(end rounds.SeasonEnd) *seasonsv1.SeasonClosed {
	table := make([]*seasonsv1.SeasonPlace, 0, len(end.Table))
	for _, placed := range end.Table {
		table = append(table, &seasonsv1.SeasonPlace{
			Country:      string(placed.Country),
			Rank:         placed.Rank,
			Points:       uint32(placed.Points), //nolint:gosec // 25 a day for a few weeks.
			RoundsWon:    placed.RoundsWon,
			FinalePoints: uint32(placed.FinalePoints), //nolint:gosec // 75 a finale.
		})
	}
	return &seasonsv1.SeasonClosed{
		Season:  uint32(end.Season),
		EndedAt: timestamppb.New(end.EndedAt),
		Table:   table,
	}
}

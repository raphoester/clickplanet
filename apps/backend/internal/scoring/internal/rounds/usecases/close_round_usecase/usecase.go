package close_round_usecase

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Rounds interface {
	Number(ctx context.Context, round rounds.Round) (uint32, error)
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

func (u *UseCase) Execute(ctx context.Context, round rounds.Round, snapshot rounds.Snapshot) (rounds.Closed, error) {
	number, err := u.store.Number(ctx, round)
	if err != nil {
		return rounds.Closed{}, fmt.Errorf("failed to number the round: %w", err)
	}

	closing := round.Closed(number, snapshot)
	err = u.transactioner.InTx(ctx, func(ctx context.Context) error {
		closed, err := u.store.Close(ctx, round, closing.Results)
		if err != nil {
			return fmt.Errorf("failed to close the round: %w", err)
		}
		if !closed {
			return nil
		}
		if err := u.outbox.Record(ctx, eventOf(closing)); err != nil {
			return fmt.Errorf("failed to record the round's close: %w", err)
		}
		return nil
	})
	if err != nil {
		return rounds.Closed{}, err
	}
	return closing, nil
}

func eventOf(closed rounds.Closed) *scoringv1.RoundClosed {
	results := make([]*scoringv1.RoundResult, 0, len(closed.Results))
	for _, result := range closed.Results {
		results = append(results, &scoringv1.RoundResult{
			Country: string(result.Country),
			Rank:    result.Rank,
			Points:  result.Points,
		})
	}
	return &scoringv1.RoundClosed{
		Season:  uint32(closed.Round.Season),
		Number:  closed.Number,
		Finale:  closed.Round.Finale,
		EndedAt: timestamppb.New(closed.Round.EndsAt),
		Results: results,
		Own:     closed.Round.Own,
		Times:   closed.Round.Times,
	}
}

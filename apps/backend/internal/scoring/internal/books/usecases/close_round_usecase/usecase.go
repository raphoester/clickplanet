package close_round_usecase

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Territory interface {
	Snapshot(ctx context.Context) (books.Snapshot, error)
}

type Rounds interface {
	Find(ctx context.Context, id books.RoundID) (books.Round, bool, error)
	Score(ctx context.Context, scored books.Scored) (bool, error)
}

type Command struct {
	Round  books.RoundID
	EndsAt time.Time
}

type UseCase struct {
	transactioner cpatomic.Transactioner
	territory     Territory
	rounds        Rounds
	outbox        cpatomic.Outbox
}

func New(transactioner cpatomic.Transactioner, territory Territory, rounds Rounds, outbox cpatomic.Outbox) *UseCase {
	return &UseCase{transactioner: transactioner, territory: territory, rounds: rounds, outbox: outbox}
}

// Execute scores the round from the map now, and answers whether it did: a round never opened, or scored before, is
// not scored.
func (u *UseCase) Execute(ctx context.Context, command Command) (bool, error) {
	round, found, err := u.rounds.Find(ctx, command.Round)
	if err != nil || !found || round.Scored {
		return false, err //nolint:wrapcheck // the store says what failed.
	}
	snapshot, err := u.territory.Snapshot(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to read the map to score round %s: %w", command.Round, err)
	}

	scored := round.ScoredAt(command.EndsAt, snapshot)
	done := false
	err = u.transactioner.InTx(ctx, func(ctx context.Context) error {
		if done, err = u.rounds.Score(ctx, scored); err != nil || !done {
			return err //nolint:wrapcheck // the store says what failed.
		}
		if err := u.outbox.Record(ctx, eventOf(scored)); err != nil {
			return fmt.Errorf("failed to record round %s scored: %w", command.Round, err)
		}
		return nil
	})
	if err != nil {
		return false, err //nolint:wrapcheck // each step named what failed.
	}
	return done, nil
}

func eventOf(scored books.Scored) *scoringv1.RoundScored {
	results := make([]*scoringv1.RoundResult, 0, len(scored.Results))
	for _, result := range scored.Results {
		results = append(results, &scoringv1.RoundResult{
			Country: string(result.Country),
			Rank:    result.Rank,
			Points:  result.Points,
			Share:   result.Share,
		})
	}
	return &scoringv1.RoundScored{
		RoundId: string(scored.Round.ID),
		BookId:  string(scored.Round.Book),
		EndedAt: timestamppb.New(scored.Round.EndsAt),
		Results: results,
	}
}

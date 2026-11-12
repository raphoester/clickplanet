package close_round_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Territory interface {
	Snapshot(ctx context.Context) (books.Snapshot, error)
}

type Rounds interface {
	Find(ctx context.Context, id books.RoundID) (books.Round, bool, error)
	StartOf(ctx context.Context, id books.RoundID) (books.Start, error)
	Fights(ctx context.Context, id books.RoundID) (map[books.Country]uint32, error)
	Score(ctx context.Context, scored books.Scored) (bool, error)
	Lock(ctx context.Context, id books.RoundID) error
	Missed(ctx context.Context, id books.RoundID) (books.MissedRound, bool, error)
	KeepMissed(ctx context.Context, missed books.MissedRound) error
}

type Command struct {
	Round  books.RoundID
	EndsAt time.Time
}

type UseCase struct {
	transactioner cpatomic.Transactioner
	territory     Territory
	rounds        Rounds
}

func New(transactioner cpatomic.Transactioner, territory Territory, rounds Rounds) *UseCase {
	return &UseCase{transactioner: transactioner, territory: territory, rounds: rounds}
}

// Execute scores the round from the map now, and answers the end it was scored at: a round scored before is not scored
// again, and one never opened is kept missed, at the end of the first close of it.
func (u *UseCase) Execute(ctx context.Context, command Command) (books.Ending, error) {
	var (
		round  books.Round
		open   bool
		ending books.Ending
	)
	err := u.transactioner.InTx(ctx, func(ctx context.Context) error {
		if err := u.rounds.Lock(ctx, command.Round); err != nil {
			return err //nolint:wrapcheck // the store says what failed.
		}
		var err error
		if round, open, err = u.rounds.Find(ctx, command.Round); err != nil || open {
			return err //nolint:wrapcheck // the store says what failed.
		}
		ending, err = u.miss(ctx, command)
		return err
	})
	if err != nil || !open {
		return ending, err //nolint:wrapcheck // each step named what failed.
	}
	if !round.Scored {
		if round, err = u.score(ctx, round, command.EndsAt); err != nil {
			return books.Ending{}, err
		}
	}
	return round.Ending(), nil
}

func (u *UseCase) miss(ctx context.Context, command Command) (books.Ending, error) {
	missed, found, err := u.rounds.Missed(ctx, command.Round)
	if err != nil {
		return books.Ending{}, err //nolint:wrapcheck // the store says what failed.
	}
	if !found {
		missed = books.NewMissedRound(command.Round, command.EndsAt)
		if err := u.rounds.KeepMissed(ctx, missed); err != nil {
			return books.Ending{}, fmt.Errorf("failed to keep round %s missed: %w", command.Round, err)
		}
	}
	return missed.Ending(), nil
}

func (u *UseCase) score(ctx context.Context, round books.Round, endsAt time.Time) (books.Round, error) {
	start, err := u.rounds.StartOf(ctx, round.ID)
	if err != nil {
		return books.Round{}, err //nolint:wrapcheck // the store says what failed.
	}
	snapshot, err := u.territory.Snapshot(ctx)
	if err != nil {
		return books.Round{}, fmt.Errorf("failed to read the map to score round %s: %w", round.ID, err)
	}
	if snapshot.Fights, err = u.rounds.Fights(ctx, round.ID); err != nil {
		return books.Round{}, fmt.Errorf("failed to read the tiles taken in round %s: %w", round.ID, err)
	}
	scored := round.ScoredAt(endsAt, start, snapshot)
	done, err := u.rounds.Score(ctx, scored)
	if err != nil {
		return books.Round{}, fmt.Errorf("failed to score round %s: %w", round.ID, err)
	}
	if done {
		return scored.Round, nil
	}
	again, _, err := u.rounds.Find(ctx, round.ID)
	return again, err //nolint:wrapcheck // the store says what failed.
}

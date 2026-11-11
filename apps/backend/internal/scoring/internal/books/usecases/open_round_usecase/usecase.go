package open_round_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Books interface {
	Find(ctx context.Context, id books.BookID) (books.Book, bool, error)
}

type Rounds interface {
	Find(ctx context.Context, id books.RoundID) (books.Round, bool, error)
	Of(ctx context.Context, books []books.BookID) ([]books.Round, error)
	Lock(ctx context.Context, id books.RoundID) error
	Missed(ctx context.Context, id books.RoundID) (books.MissedRound, bool, error)
	Open(ctx context.Context, round books.Round, start books.Start) (bool, error)
}

type Territory interface {
	Snapshot(ctx context.Context) (books.Snapshot, error)
}

var ErrBookNotOpen = errors.New("the round's book is not open")

type Command struct {
	Round    books.RoundID
	Book     books.BookID
	Times    uint32
	Ranking  books.Ranking
	StartsAt time.Time
	// FromNothing is a round told to count from nothing.
	FromNothing bool
}

func New(transactioner cpatomic.Transactioner, books Books, rounds Rounds, territory Territory) *UseCase {
	return &UseCase{transactioner: transactioner, books: books, rounds: rounds, territory: territory}
}

type UseCase struct {
	transactioner cpatomic.Transactioner
	books         Books
	rounds        Rounds
	territory     Territory
}

// Execute answers whether it opened the round: one kept before is left as it is, and one missed or of no open book
// is refused.
func (u *UseCase) Execute(ctx context.Context, command Command) (bool, error) {
	round, err := books.NewRound(command.Round, command.Book, command.Times, command.Ranking, command.StartsAt)
	if err != nil {
		return false, err //nolint:wrapcheck // the round says why.
	}
	if _, found, err := u.rounds.Find(ctx, command.Round); err != nil || found {
		return false, err //nolint:wrapcheck // the store says what failed.
	}
	book, found, err := u.books.Find(ctx, command.Book)
	if err != nil {
		return false, err //nolint:wrapcheck // the store says what failed.
	}
	if !found || book.Closed {
		return false, fmt.Errorf("%w: round %s: %w: %s", books.ErrRefused, command.Round, ErrBookNotOpen, command.Book)
	}
	start, err := u.startOf(ctx, round, command.FromNothing)
	if err != nil {
		return false, err
	}

	opened := false
	err = u.transactioner.InTx(ctx, func(ctx context.Context) error {
		if err := u.rounds.Lock(ctx, command.Round); err != nil {
			return err //nolint:wrapcheck // the store says what failed.
		}
		_, missed, err := u.rounds.Missed(ctx, command.Round)
		if err != nil {
			return err //nolint:wrapcheck // the store says what failed.
		}
		if missed {
			return fmt.Errorf("%w: round %s: %w", books.ErrRefused, command.Round, books.ErrMissed)
		}
		if opened, err = u.rounds.Open(ctx, round, start); err != nil {
			return fmt.Errorf("failed to open round %s: %w", command.Round, err)
		}
		return nil
	})
	return opened, err //nolint:wrapcheck // each step named what failed.
}

func (u *UseCase) startOf(ctx context.Context, round books.Round, fromNothing bool) (books.Start, error) {
	before, err := u.rounds.Of(ctx, []books.BookID{round.Book})
	if err != nil {
		return nil, fmt.Errorf("failed to read the rounds of book %s: %w", round.Book, err)
	}
	if books.CountsFromNothing(fromNothing, before) {
		return books.Start{}, nil
	}
	snapshot, err := u.territory.Snapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read the map to open round %s: %w", round.ID, err)
	}
	return books.Start(snapshot.Held), nil
}

package close_book_usecase

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Books interface {
	Find(ctx context.Context, id books.BookID) (books.Book, bool, error)
	Close(ctx context.Context, book books.Book) (bool, error)
}

type Rounds interface {
	ScoredIn(ctx context.Context, book books.BookID) ([]books.Scored, error)
}

type Grants interface {
	Since(ctx context.Context, at time.Time) ([]books.Grant, error)
}

type Command struct {
	Book     books.BookID
	EndsAt   time.Time
	Tiebreak books.RoundID
}

type UseCase struct {
	transactioner cpatomic.Transactioner
	books         Books
	rounds        Rounds
	grants        Grants
	outbox        cpatomic.Outbox
}

func New(transactioner cpatomic.Transactioner, books Books, rounds Rounds, grants Grants, outbox cpatomic.Outbox) *UseCase {
	return &UseCase{transactioner: transactioner, books: books, rounds: rounds, grants: grants, outbox: outbox}
}

// Execute ranks the book from its rounds scored and its grants and closes it, and answers whether it did: a book never opened, or
// closed before, is not.
func (u *UseCase) Execute(ctx context.Context, command Command) (bool, error) {
	book, found, err := u.books.Find(ctx, command.Book)
	if err != nil || !found || book.Closed {
		return false, err //nolint:wrapcheck // the store says what failed.
	}
	scored, err := u.rounds.ScoredIn(ctx, book.ID)
	if err != nil {
		return false, fmt.Errorf("failed to read the rounds book %s scored: %w", book.ID, err)
	}
	grants, err := u.grants.Since(ctx, book.StartsAt)
	if err != nil {
		return false, fmt.Errorf("failed to read the grants of book %s: %w", book.ID, err)
	}

	closed := book.ClosedAt(command.EndsAt, command.Tiebreak)
	done := false
	err = u.transactioner.InTx(ctx, func(ctx context.Context) error {
		if done, err = u.books.Close(ctx, closed); err != nil || !done {
			return err //nolint:wrapcheck // the store says what failed.
		}
		if err := u.outbox.Record(ctx, eventOf(closed, closed.Table(scored, grants))); err != nil {
			return fmt.Errorf("failed to record book %s closed: %w", book.ID, err)
		}
		return nil
	})
	if err != nil {
		return false, err //nolint:wrapcheck // each step named what failed.
	}
	return done, nil
}

func eventOf(book books.Book, table []books.Placed) *scoringv1.BookClosed {
	places := make([]*scoringv1.Place, 0, len(table))
	for _, placed := range table {
		places = append(places, &scoringv1.Place{
			Country:        string(placed.Country),
			Rank:           placed.Rank,
			Points:         uint32(placed.Points), //nolint:gosec // 25 a day for a few weeks.
			RoundsWon:      placed.RoundsWon,
			TiebreakPoints: uint32(placed.TiebreakPoints), //nolint:gosec // 250 a round at most.
		})
	}
	return &scoringv1.BookClosed{BookId: string(book.ID), EndedAt: timestamppb.New(book.EndsAt), Table: places}
}

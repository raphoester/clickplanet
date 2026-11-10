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

type Command struct {
	Book     books.BookID
	EndsAt   time.Time
	Tiebreak books.RoundID
}

type UseCase struct {
	transactioner cpatomic.Transactioner
	books         Books
	outbox        cpatomic.Outbox
}

func New(transactioner cpatomic.Transactioner, books Books, outbox cpatomic.Outbox) *UseCase {
	return &UseCase{transactioner: transactioner, books: books, outbox: outbox}
}

// Execute closes the book, and answers whether it did: a book closed before is not closed again, and one never opened
// is not closed and is told missed, at each close of it.
func (u *UseCase) Execute(ctx context.Context, command Command) (bool, error) {
	book, found, err := u.books.Find(ctx, command.Book)
	if err != nil {
		return false, err //nolint:wrapcheck // the store says what failed.
	}
	if !found {
		return false, u.miss(ctx, command)
	}
	if book.Closed {
		return false, nil
	}

	closed := book.ClosedAt(command.EndsAt, command.Tiebreak)
	done := false
	err = u.transactioner.InTx(ctx, func(ctx context.Context) error {
		if done, err = u.books.Close(ctx, closed); err != nil || !done {
			return err //nolint:wrapcheck // the store says what failed.
		}
		if err := u.outbox.Record(ctx, &scoringv1.BookClosed{
			BookId: string(closed.ID), EndedAt: timestamppb.New(closed.EndsAt),
		}); err != nil {
			return fmt.Errorf("failed to record book %s closed: %w", book.ID, err)
		}
		return nil
	})
	if err != nil {
		return false, err //nolint:wrapcheck // each step named what failed.
	}
	return done, nil
}

func (u *UseCase) miss(ctx context.Context, command Command) error {
	missed := &scoringv1.BookMissed{BookId: string(command.Book), EndedAt: timestamppb.New(command.EndsAt)}
	return u.transactioner.InTx(ctx, func(ctx context.Context) error { //nolint:wrapcheck // the outbox names the event.
		if err := u.outbox.Record(ctx, missed); err != nil {
			return fmt.Errorf("failed to record book %s missed: %w", command.Book, err)
		}
		return nil
	})
}

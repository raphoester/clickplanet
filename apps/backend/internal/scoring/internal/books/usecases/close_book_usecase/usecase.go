package close_book_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Books interface {
	Find(ctx context.Context, id books.BookID) (books.Book, bool, error)
	Close(ctx context.Context, book books.Book) (bool, error)
	Lock(ctx context.Context, id books.BookID) error
	Missed(ctx context.Context, id books.BookID) (books.MissedBook, bool, error)
	KeepMissed(ctx context.Context, missed books.MissedBook) error
}

type Command struct {
	Book     books.BookID
	EndsAt   time.Time
	Tiebreak books.RoundID
}

type UseCase struct {
	transactioner cpatomic.Transactioner
	books         Books
}

func New(transactioner cpatomic.Transactioner, books Books) *UseCase {
	return &UseCase{transactioner: transactioner, books: books}
}

// Execute closes the book, and answers the end it closed at: a book closed before is not closed again, and one never
// opened is kept missed, at the end of the first close of it.
func (u *UseCase) Execute(ctx context.Context, command Command) (books.Ending, error) {
	var ending books.Ending
	err := u.transactioner.InTx(ctx, func(ctx context.Context) error {
		if err := u.books.Lock(ctx, command.Book); err != nil {
			return err //nolint:wrapcheck // the store says what failed.
		}
		book, open, err := u.books.Find(ctx, command.Book)
		if err != nil {
			return err //nolint:wrapcheck // the store says what failed.
		}
		if !open {
			ending, err = u.miss(ctx, command)
			return err
		}
		if !book.Closed {
			book = book.ClosedAt(command.EndsAt, command.Tiebreak)
			if _, err := u.books.Close(ctx, book); err != nil {
				return fmt.Errorf("failed to close book %s: %w", command.Book, err)
			}
		}
		ending = book.Ending()
		return nil
	})
	return ending, err //nolint:wrapcheck // each step named what failed.
}

func (u *UseCase) miss(ctx context.Context, command Command) (books.Ending, error) {
	missed, found, err := u.books.Missed(ctx, command.Book)
	if err != nil {
		return books.Ending{}, err //nolint:wrapcheck // the store says what failed.
	}
	if !found {
		missed = books.NewMissedBook(command.Book, command.EndsAt)
		if err := u.books.KeepMissed(ctx, missed); err != nil {
			return books.Ending{}, fmt.Errorf("failed to keep book %s missed: %w", command.Book, err)
		}
	}
	return missed.Ending(), nil
}

package open_book_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Books interface {
	Lock(ctx context.Context, id books.BookID) error
	Missed(ctx context.Context, id books.BookID) (books.MissedBook, bool, error)
	Open(ctx context.Context, book books.Book) (bool, error)
}

type Command struct {
	Book     books.BookID
	StartsAt time.Time
}

func New(transactioner cpatomic.Transactioner, store Books) *UseCase {
	return &UseCase{transactioner: transactioner, store: store}
}

type UseCase struct {
	transactioner cpatomic.Transactioner
	store         Books
}

// Execute answers whether it opened the book: one kept before is left as it is, and one a close missed is refused.
func (u *UseCase) Execute(ctx context.Context, command Command) (bool, error) {
	opened := false
	err := u.transactioner.InTx(ctx, func(ctx context.Context) error {
		if err := u.store.Lock(ctx, command.Book); err != nil {
			return err //nolint:wrapcheck // the store says what failed.
		}
		_, missed, err := u.store.Missed(ctx, command.Book)
		if err != nil {
			return err //nolint:wrapcheck // the store says what failed.
		}
		if missed {
			return fmt.Errorf("%w: book %s: %w", books.ErrRefused, command.Book, books.ErrMissed)
		}
		if opened, err = u.store.Open(ctx, books.NewBook(command.Book, command.StartsAt)); err != nil {
			return fmt.Errorf("failed to open book %s: %w", command.Book, err)
		}
		return nil
	})
	return opened, err //nolint:wrapcheck // each step named what failed.
}

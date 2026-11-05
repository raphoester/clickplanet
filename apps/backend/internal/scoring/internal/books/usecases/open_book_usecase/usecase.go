package open_book_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
)

type Books interface {
	Find(ctx context.Context, id books.BookID) (books.Book, bool, error)
	Open(ctx context.Context, book books.Book) (bool, error)
}

type Command struct {
	Book     books.BookID
	StartsAt time.Time
}

func New(store Books) *UseCase {
	return &UseCase{store: store}
}

type UseCase struct {
	store Books
}

// Execute answers whether it opened the book: one kept before is left as it is.
func (u *UseCase) Execute(ctx context.Context, command Command) (bool, error) {
	if _, found, err := u.store.Find(ctx, command.Book); err != nil || found {
		return false, err //nolint:wrapcheck // the store says what failed.
	}
	opened, err := u.store.Open(ctx, books.NewBook(command.Book, command.StartsAt))
	if err != nil {
		return false, fmt.Errorf("failed to open book %s: %w", command.Book, err)
	}
	return opened, nil
}

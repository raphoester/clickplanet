package record_take_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings"
)

type Books interface {
	All(ctx context.Context) ([]books.Book, error)
}

type Recorder interface {
	RecordTake(ctx context.Context, book books.BookID, take standings.Take) error
}

type UseCase struct {
	books    Books
	recorder Recorder
}

func New(books Books, recorder Recorder) *UseCase {
	return &UseCase{books: books, recorder: recorder}
}

// Execute counts the take in every book open at its time.
func (u *UseCase) Execute(ctx context.Context, take standings.Take) error {
	all, err := u.books.All(ctx)
	if err != nil {
		return fmt.Errorf("failed to read the books: %w", err)
	}
	for _, book := range all {
		if !book.Holds(take.At) {
			continue
		}
		if err := u.recorder.RecordTake(ctx, book.ID, take); err != nil {
			return fmt.Errorf("failed to count the take in book %s: %w", book.ID, err)
		}
	}
	return nil
}

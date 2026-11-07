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
	RecordTakes(ctx context.Context, book books.BookID, account standings.AccountID, takes []standings.Take) error
}

type UseCase struct {
	books    Books
	recorder Recorder
}

func New(books Books, recorder Recorder) *UseCase {
	return &UseCase{books: books, recorder: recorder}
}

// Execute counts each take in every book open at its time, one write per book and account: the books are read once
// for the whole batch, and an enclose of a continent is one write, not one per tile.
func (u *UseCase) Execute(ctx context.Context, takes []standings.Take) error {
	if len(takes) == 0 {
		return nil
	}
	all, err := u.books.All(ctx)
	if err != nil {
		return fmt.Errorf("failed to read the books: %w", err)
	}
	for _, book := range all {
		var told []standings.AccountID
		held := map[standings.AccountID][]standings.Take{}
		for _, take := range takes {
			if !book.Holds(take.At) {
				continue
			}
			if _, known := held[take.Account]; !known {
				told = append(told, take.Account)
			}
			held[take.Account] = append(held[take.Account], take)
		}
		for _, account := range told {
			if err := u.recorder.RecordTakes(ctx, book.ID, account, held[account]); err != nil {
				return fmt.Errorf("failed to count the takes of %s in book %s: %w", account, book.ID, err)
			}
		}
	}
	return nil
}

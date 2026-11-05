package count_take_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
)

type Books interface {
	All(ctx context.Context) ([]books.Book, error)
}

type Rounds interface {
	Of(ctx context.Context, books []books.BookID) ([]books.Round, error)
}

type Takes interface {
	Count(ctx context.Context, rounds []books.RoundID, take books.Take) error
}

func New(books Books, rounds Rounds, takes Takes) *UseCase {
	return &UseCase{books: books, rounds: rounds, takes: takes}
}

// UseCase counts each take in every round open at its time, of every book open at its time, for their numbers.
type UseCase struct {
	books  Books
	rounds Rounds
	takes  Takes
}

func (u *UseCase) Execute(ctx context.Context, take books.Take) error {
	all, err := u.books.All(ctx)
	if err != nil {
		return fmt.Errorf("failed to read the books: %w", err)
	}
	var holding []books.BookID
	for _, book := range all {
		if book.Holds(take.At) {
			holding = append(holding, book.ID)
		}
	}
	if len(holding) == 0 {
		return nil
	}

	of, err := u.rounds.Of(ctx, holding)
	if err != nil {
		return fmt.Errorf("failed to read the rounds of the books: %w", err)
	}
	var counting []books.RoundID
	for _, round := range of {
		if round.Holds(take.At) {
			counting = append(counting, round.ID)
		}
	}
	if len(counting) == 0 {
		return nil
	}
	if err := u.takes.Count(ctx, counting, take); err != nil {
		return fmt.Errorf("failed to count the take: %w", err)
	}
	return nil
}

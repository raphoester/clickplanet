package count_take_usecase

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
)

type Books interface {
	All(ctx context.Context) ([]books.Book, error)
}

type Rounds interface {
	Of(ctx context.Context, books []books.BookID) ([]books.Round, error)
}

type Takes interface {
	Count(ctx context.Context, held map[books.RoundID][]books.Take) error
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

// Execute reads the books and their rounds once for the whole batch, and counts it in one write.
func (u *UseCase) Execute(ctx context.Context, takes []books.Take) error {
	if len(takes) == 0 {
		return nil
	}
	all, err := u.books.All(ctx)
	if err != nil {
		return fmt.Errorf("failed to read the books: %w", err)
	}
	open := map[books.BookID]books.Book{}
	var holding []books.BookID
	for _, book := range all {
		if slices.ContainsFunc(takes, func(take books.Take) bool { return book.Holds(take.At) }) {
			open[book.ID] = book
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
	held := map[books.RoundID][]books.Take{}
	var (
		at       time.Time
		counting []books.RoundID
	)
	for i, take := range takes {
		// The tiles of one event share its time: the rounds holding it are found once.
		if i == 0 || !take.At.Equal(at) {
			at, counting = take.At, holdingAt(of, open, take.At)
		}
		for _, round := range counting {
			held[round] = append(held[round], take)
		}
	}
	if len(held) == 0 {
		return nil
	}
	if err := u.takes.Count(ctx, held); err != nil {
		return fmt.Errorf("failed to count the takes: %w", err)
	}
	return nil
}

// holdingAt is the rounds that count a take at a time: open at it, of a book open at it.
func holdingAt(rounds []books.Round, open map[books.BookID]books.Book, at time.Time) []books.RoundID {
	var holding []books.RoundID
	for _, round := range rounds {
		if book, found := open[round.Book]; found && book.Holds(at) && round.Holds(at) {
			holding = append(holding, round.ID)
		}
	}
	return holding
}

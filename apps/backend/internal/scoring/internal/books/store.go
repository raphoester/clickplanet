package books

import (
	"context"
	"time"
)

type Books interface {
	Find(ctx context.Context, id BookID) (Book, bool, error)
	All(ctx context.Context) ([]Book, error)
	// Open answers whether this call opened the book: one kept before is left as it is.
	Open(ctx context.Context, book Book) (bool, error)
	// Close answers whether this call closed the book: one closed before is left as it is.
	Close(ctx context.Context, book Book) (bool, error)
	// Lock holds the id until the transaction ctx carries ends: call it in one.
	Lock(ctx context.Context, id BookID) error
	Missed(ctx context.Context, id BookID) (MissedBook, bool, error)
	KeepMissed(ctx context.Context, missed MissedBook) error
}

type Rounds interface {
	Find(ctx context.Context, id RoundID) (Round, bool, error)
	// Of is every round of the books, in the order they started.
	Of(ctx context.Context, books []BookID) ([]Round, error)
	// Open answers whether this call opened the round, kept with its start: one kept before is left as it is.
	Open(ctx context.Context, round Round, start Start) (bool, error)
	// StartOf is what each country held when the round opened: empty for one opened before starts were kept.
	StartOf(ctx context.Context, id RoundID) (Start, error)
	// Score answers whether this call scored the round: one scored before is left as it is.
	Score(ctx context.Context, scored Scored) (bool, error)
	// ScoredIn is every round of the book scored, in the order they ended.
	ScoredIn(ctx context.Context, book BookID) ([]Scored, error)
	// Lock holds the id until the transaction ctx carries ends: call it in one.
	Lock(ctx context.Context, id RoundID) error
	Missed(ctx context.Context, id RoundID) (MissedRound, bool, error)
	KeepMissed(ctx context.Context, missed MissedRound) error
}

type Grants interface {
	// Keep answers whether this call kept the grant: one kept before is left as it is.
	Keep(ctx context.Context, grant Grant) (bool, error)
	// Since is every grant from a time on, in the order they were granted.
	Since(ctx context.Context, at time.Time) ([]Grant, error)
}

// Takes counts the tiles taken during each round, for its numbers.
type Takes interface {
	// Count counts the takes each round holds, given in ledger order, each once and in one write: a take told again
	// changes nothing.
	Count(ctx context.Context, held map[RoundID][]Take) error
	ForgetAccount(ctx context.Context, account AccountID) error
}

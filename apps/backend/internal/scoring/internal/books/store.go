package books

import "context"

type Books interface {
	Find(ctx context.Context, id BookID) (Book, bool, error)
	All(ctx context.Context) ([]Book, error)
	// Open answers whether this call opened the book: one kept before is left as it is.
	Open(ctx context.Context, book Book) (bool, error)
	// Close answers whether this call closed the book: one closed before is left as it is.
	Close(ctx context.Context, book Book) (bool, error)
}

type Rounds interface {
	Find(ctx context.Context, id RoundID) (Round, bool, error)
	// Of is every round of the books, in the order they started.
	Of(ctx context.Context, books []BookID) ([]Round, error)
	// Open answers whether this call opened the round: one kept before is left as it is.
	Open(ctx context.Context, round Round) (bool, error)
	// Score answers whether this call scored the round: one scored before is left as it is.
	Score(ctx context.Context, scored Scored) (bool, error)
	// ScoredIn is every round of the book scored, in the order they ended.
	ScoredIn(ctx context.Context, book BookID) ([]Scored, error)
}

// Takes counts the tiles taken during each round, for its numbers.
type Takes interface {
	// Count counts a take in each round, once: a take told again changes nothing.
	Count(ctx context.Context, rounds []RoundID, take Take) error
	ForgetAccount(ctx context.Context, account AccountID) error
}

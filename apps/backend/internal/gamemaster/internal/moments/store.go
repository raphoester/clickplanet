package moments

import (
	"context"
	"time"
)

// ScoredRound is a round as scoring told it scored.
type ScoredRound struct {
	Round   RoundID
	Book    BookID
	EndedAt time.Time
}

// ClosedBook is a book as scoring told it closed.
type ClosedBook struct {
	Book    BookID
	EndedAt time.Time
}

// Store keeps what scoring told, each once.
type Store interface {
	// KeepRound is true when the round was not kept yet.
	KeepRound(ctx context.Context, scored ScoredRound) (bool, error)
	// KeepBook is true when the book was not kept yet.
	KeepBook(ctx context.Context, closed ClosedBook) (bool, error)
	KeepOpened(ctx context.Context, round RoundID) error
	Opened(ctx context.Context) ([]RoundID, error)
}

// Events tells the other modules a moment.
type Events interface {
	Record(ctx context.Context, moment Moment) error
}

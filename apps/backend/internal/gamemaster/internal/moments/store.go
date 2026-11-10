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

// MissedRound is a round scoring told missed: closed before it ever opened, so nothing scores it.
type MissedRound struct {
	Round   RoundID
	EndedAt time.Time
}

// MissedBook is a book scoring told missed: closed before it ever opened, so nothing ranks it.
type MissedBook struct {
	Book    BookID
	EndedAt time.Time
}

// ToldDay is a day as the gamemaster told it.
type ToldDay struct {
	Round  RoundID
	EndsAt time.Time
}

// Store keeps what scoring told, each once, each day as it was told, and each season and battle told.
type Store interface {
	// KeepRound is true when the round was not kept yet.
	KeepRound(ctx context.Context, scored ScoredRound) (bool, error)
	// KeepBook is true when the book was not kept yet.
	KeepBook(ctx context.Context, closed ClosedBook) (bool, error)
	// KeepMissedRound is true when the round was not kept missed yet.
	KeepMissedRound(ctx context.Context, missed MissedRound) (bool, error)
	// KeepMissedBook is true when the book was not kept missed yet.
	KeepMissedBook(ctx context.Context, missed MissedBook) (bool, error)
	KeepOpened(ctx context.Context, round RoundID) error
	// KeepToldSeason is true when the season was not told yet.
	KeepToldSeason(ctx context.Context, book BookID) (bool, error)
	// KeepToldDay is true when the day was not told yet, or was told with another end.
	KeepToldDay(ctx context.Context, day ToldDay) (bool, error)
	// KeepToldBattle is true when the battle was not told yet.
	KeepToldBattle(ctx context.Context, round RoundID) (bool, error)
	// Kept is every round opened, and every round and book scoring answered, in one read.
	Kept(ctx context.Context) (Kept, error)
}

// Events tells the other modules a moment.
type Events interface {
	Record(ctx context.Context, moment Moment) error
}

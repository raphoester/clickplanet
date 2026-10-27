package rounds

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar"
)

type Store interface {
	Open(ctx context.Context, round Round) error
	// Unclosed is every round ended by a time and not closed, the earliest end first, a day before a round of its
	// own that ends with it.
	Unclosed(ctx context.Context, endedBy time.Time) ([]Round, error)
	// FirstEnd is the earliest end of a round not closed; false when every round is.
	FirstEnd(ctx context.Context) (time.Time, bool, error)
	// Number is a day's place among its season's days, a round of its own's among its season's rounds of their own.
	Number(ctx context.Context, round Round) (uint32, error)
	// Close answers whether this call closed the round: one closed before writes nothing.
	Close(ctx context.Context, round Round, results []Result) (bool, error)
	ClosedIn(ctx context.Context, season calendar.Number) ([]Closed, error)
	// End answers whether this call ended the season: one ended before writes nothing.
	End(ctx context.Context, season calendar.Season) (bool, error)
	Ended(ctx context.Context, season calendar.Number) (bool, error)
}

// Takes counts the tiles taken during each round of its own, for its numbers.
type Takes interface {
	// Count counts a take in every round of its own whose window holds it, once: a take told again changes nothing.
	Count(ctx context.Context, take Take) error
	ForgetAccount(ctx context.Context, account AccountID) error
}

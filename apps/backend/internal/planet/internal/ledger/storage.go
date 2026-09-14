package ledger

import "time"

// Storage keeps one take per tile at most.
type Storage interface {
	Last(tile uint32) (Taking, bool)
	// Put replaces the tile's last take.
	Put(taking Taking)
	PaintedWith(country string) []Taking
	TakenBy(scope string) []Taking
	// Forget drops these takes, unless the tile was taken again since.
	Forget(takings []Taking)
	ForgetBefore(cutoff time.Time)
}

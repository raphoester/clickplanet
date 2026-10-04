package clicks

import (
	"context"
	"time"
)

// AllegianceStorage is every tally as it is kept. Each reader declares the part it needs; this is the whole of it,
// which AllegianceStorageContractSuite pins.
type AllegianceStorage interface {
	// Allegiances reads the tallies under keys. A key with no tally is absent.
	Allegiances(ctx context.Context, keys ...AllegianceKey) (map[AllegianceKey]Allegiance, error)

	// SaveAllegiances writes every tally, replacing what was under its key.
	SaveAllegiances(ctx context.Context, tallies map[AllegianceKey]Allegiance) error

	// DeleteAllegiancesBefore deletes the tallies whose last take is before cutoff, and says how many went.
	DeleteAllegiancesBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

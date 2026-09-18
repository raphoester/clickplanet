package clicks

import "context"

// TileStorage is the map as it is kept: who holds each tile, and the one feed of what changed.
// Every implementation runs TileStorageContractSuite.
type TileStorage interface {
	Owner(tile uint32) (string, bool)
	Share(country string) float64
	Held(country string) int
	StateBatchDense(start uint32, end uint32) (DenseBatch, error)

	// Set publishes an update only when the tile changes hands.
	Set(ctx context.Context, tile uint32, value string) error
	// Clear empties the blast's tiles and publishes it once, holding only the tiles that were owned.
	Clear(ctx context.Context, blast Blast) (Blast, error)
	// Reassign moves up to limit of from's tiles to to, scanning from start; next is 0 once the map is done.
	Reassign(ctx context.Context, from, to string, start uint32, limit int) (next uint32, moved int, err error)
	// Restore applies each restoration whose tile still holds From.
	Restore(ctx context.Context, restorations []Restoration) (int, error)

	// Subscribe is one feed per call, closed when ctx ends.
	Subscribe(ctx context.Context) (<-chan Change, error)
}

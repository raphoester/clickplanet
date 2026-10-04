package clicks

import "context"

type TileStorage interface {
	Owner(tile uint32) (string, bool)
	Share(country string) float64
	Held(country string) int
	Holdings() []Holding
	StateBatchDense(start uint32, end uint32) (DenseBatch, error)

	Set(ctx context.Context, tile uint32, value string) error
	Click(ctx context.Context, tile uint32, value string) error
	Clear(ctx context.Context, blast Blast) (Blast, error)
	Reassign(ctx context.Context, from, to string, start uint32, limit int) (next uint32, moved int, err error)
	Restore(ctx context.Context, restorations []Restoration) (int, error)

	Subscribe(ctx context.Context) (<-chan Change, error)
}

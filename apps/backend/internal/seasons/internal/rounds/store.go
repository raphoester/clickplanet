package rounds

import (
	"context"
	"time"
)

type Store interface {
	RecordSnapshot(ctx context.Context, round Round, snapshot Snapshot) error
	Unclosed(ctx context.Context, endedBy time.Time) ([]Round, error)
	Held(ctx context.Context, round Round) (map[Country]uint64, error)
	Close(ctx context.Context, round Round, results []Result) error
}

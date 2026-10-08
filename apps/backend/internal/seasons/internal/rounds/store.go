package rounds

import (
	"context"
	"time"
)

type Store interface {
	RecordCensus(ctx context.Context, round Round, census Census) error
	Unclosed(ctx context.Context, endedBy time.Time) ([]Round, error)
	Held(ctx context.Context, round Round) (map[Country]uint64, error)
	Close(ctx context.Context, round Round, results []Result) error
}

package rounds

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
)

type Store interface {
	RecordSnapshot(ctx context.Context, round Round, snapshot Snapshot) error
	Unclosed(ctx context.Context, endedBy time.Time) ([]Round, error)
	Held(ctx context.Context, round Round) (map[Country]uint64, error)
	Number(ctx context.Context, round Round) (uint32, error)
	Close(ctx context.Context, round Round, results []Result) (bool, error)
	ClosedIn(ctx context.Context, season calendar.Number) ([]Closed, error)
}

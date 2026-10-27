package rounds

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
)

type Store interface {
	Open(ctx context.Context, round Round) error
	Unclosed(ctx context.Context, endedBy time.Time) ([]Round, error)
	Number(ctx context.Context, round Round) (uint32, error)
	Close(ctx context.Context, round Round, results []Result) (bool, error)
	ClosedIn(ctx context.Context, season calendar.Number) ([]Closed, error)
}

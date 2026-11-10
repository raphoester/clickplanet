//go:build testing

package stats

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type RecordedPause struct {
	clock  *cptime.FixedClock
	Waited []time.Duration
}

func NewRecordedPause(clock *cptime.FixedClock) *RecordedPause {
	return &RecordedPause{clock: clock}
}

var _ Pause = (*RecordedPause)(nil)

func (p *RecordedPause) Wait(_ context.Context, d time.Duration) error {
	p.Waited = append(p.Waited, d)
	p.clock.Advance(d)
	return nil
}

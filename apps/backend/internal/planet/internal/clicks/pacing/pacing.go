// Package pacing spreads an operator's bulk change over time, so each batch of updates fits what an open stream can buffer.
package pacing

import (
	"context"
	"fmt"
	"time"
)

type Pacing struct {
	Batch int
	Pause time.Duration
}

// Wait sleeps one pause, and says whether the caller should stop.
func (p Pacing) Wait(ctx context.Context) error {
	if p.Pause > 0 {
		timer := time.NewTimer(p.Pause)
		defer timer.Stop()

		select {
		case <-ctx.Done():
		case <-timer.C:
		}
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("interrupted: %w", err)
	}

	return nil
}

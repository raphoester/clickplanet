package clicks

import (
	"context"
	"fmt"
	"time"
)

type Pacing struct {
	Batch int
	Pause time.Duration
}

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

package posts

import (
	"context"
	"time"
)

type SystemPause struct{}

var _ Pause = SystemPause{}

func (SystemPause) Wait(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err() //nolint:wrapcheck // the context's own error says why the wait ended.
	}
}

package posts

import (
	"context"
	"math/rand/v2"
	"time"
)

type SystemDice struct{}

var _ Dice = SystemDice{}

func (SystemDice) IntN(n int) int {
	return rand.IntN(n) //nolint:gosec // the words of a post, not a secret.
}

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

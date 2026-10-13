package replays

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrInvalidRequest = errors.New("the replay asked for is invalid")

// Until zero is now.
type Request struct {
	Span  time.Duration
	Until time.Time
}

func (r Request) Window(now time.Time) (Window, error) {
	if r.Span <= 0 {
		return Window{}, fmt.Errorf("%w: it spans no time", ErrInvalidRequest)
	}

	until := r.Until
	if until.IsZero() {
		until = now
	}
	return Window{Since: until.Add(-r.Span), Until: until}, nil
}

type Window struct {
	Since time.Time
	Until time.Time
}

type Replay struct {
	// A planet.v1.GetReplayResponse in JSON, as the backend answered it.
	Document []byte
	Events   int
	Window   Window
}

type Source interface {
	Replay(ctx context.Context, window Window) (Replay, error)
}

type Store interface {
	Save(ctx context.Context, replay Replay) error
}

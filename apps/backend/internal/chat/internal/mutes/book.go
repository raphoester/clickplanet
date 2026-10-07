package mutes

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Reader interface {
	Mute(ctx context.Context, caller Caller, at time.Time) (Mute, error)
}

const readTimeout = 5 * time.Second

func NewBook(mutes Reader, clock cptime.Clock) Book {
	return Book{mutes: mutes, clock: clock}
}

type Book struct {
	mutes Reader
	clock cptime.Clock
}

func (b Book) MuteError(ctx context.Context, caller Caller) error {
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	mute, err := b.mutes.Mute(ctx, caller, b.clock.Now())
	switch {
	case errors.Is(err, ErrNotMuted):
		return nil
	case err != nil:
		return fmt.Errorf("failed to read whether the caller is muted: %w", err)
	default:
		return mute.Refusal()
	}
}

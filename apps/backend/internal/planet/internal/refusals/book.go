package refusals

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store interface {
	Refusal(ctx context.Context, caller Caller) (Refusal, error)
	// Change runs under a lock on the caller, and keeps what it answers when that differs from what it was handed.
	Change(ctx context.Context, caller Caller, change func(Refusal) Refusal) error
}

const readTimeout = time.Second

func NewBook(store Store, clock cptime.Clock) Book {
	return Book{store: store, clock: clock}
}

type Book struct {
	store Store
	clock cptime.Clock
}

func (b Book) Decide(ctx context.Context, caller Caller, decision Decision) error {
	if err := b.store.Change(ctx, caller, func(refusal Refusal) Refusal {
		return refusal.Decided(decision)
	}); err != nil {
		return fmt.Errorf("failed to keep the decision on %+v: %w", caller, err)
	}
	return nil
}

// The refusal of the scope or of the account that ends last.
func (b Book) Refusal(ctx context.Context, payer clicks.Payer) (clicks.Refusal, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	now := b.clock.Now()

	var until time.Time
	for _, caller := range callersOf(payer) {
		refusal, err := b.store.Refusal(ctx, caller)
		if err != nil {
			return clicks.Refusal{}, false, fmt.Errorf("failed to read the refusal of %+v: %w", caller, err)
		}
		if refusal.Standing(now) && refusal.until.After(until) {
			until = refusal.until
		}
	}

	if until.IsZero() {
		return clicks.Refusal{}, false, nil
	}
	return clicks.NewRefusal(until), true, nil
}

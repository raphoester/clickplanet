package refusals

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store interface {
	// Refusals holds each caller ever decided on; a caller never decided on is not in it.
	Refusals(ctx context.Context, callers []Caller) (map[Caller]Refusal, error)
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
	refused, err := b.Refusals(ctx, []clicks.Payer{payer})
	if err != nil {
		return clicks.Refusal{}, false, err
	}
	refusal, ok := refused[payer]
	return refusal, ok, nil
}

// The refusal of each payer, read at once; a payer refused nothing is not in it.
func (b Book) Refusals(ctx context.Context, payers []clicks.Payer) (map[clicks.Payer]clicks.Refusal, error) {
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	callers := make([]Caller, 0, 2*len(payers))
	for _, payer := range payers {
		callers = append(callers, callersOf(payer)...)
	}
	kept, err := b.store.Refusals(ctx, callers)
	if err != nil {
		return nil, fmt.Errorf("failed to read the refusals of %d callers: %w", len(callers), err)
	}

	now := b.clock.Now()
	refused := map[clicks.Payer]clicks.Refusal{}
	for _, payer := range payers {
		var until time.Time
		for _, caller := range callersOf(payer) {
			if refusal := kept[caller]; refusal.Standing(now) && refusal.until.After(until) {
				until = refusal.until
			}
		}
		if !until.IsZero() {
			refused[payer] = clicks.NewRefusal(until)
		}
	}
	return refused, nil
}

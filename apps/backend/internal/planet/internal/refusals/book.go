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

func (b Book) Acknowledge(ctx context.Context, payer clicks.Payer) error {
	if payer.Account == "" {
		return nil
	}

	caller := Caller{Account: payer.Account}
	if err := b.store.Change(ctx, caller, func(refusal Refusal) Refusal {
		return refusal.Acknowledged(b.clock.Now())
	}); err != nil {
		return fmt.Errorf("failed to keep the warning acknowledged by %+v: %w", caller, err)
	}
	return nil
}

// The ban of the scope or of the account that ends last, or else the account's warning.
func (b Book) Refusal(ctx context.Context, payer clicks.Payer) (clicks.Refusal, bool, error) {
	standings, err := b.standings(ctx, []clicks.Payer{payer})
	if err != nil {
		return clicks.Refusal{}, false, err
	}
	refusal, refused := standings[payer].refusal()
	return refusal, refused, nil
}

// The ban of each payer, read at once; a payer banned by nothing is not in it, a warned one neither.
func (b Book) Refusals(ctx context.Context, payers []clicks.Payer) (map[clicks.Payer]clicks.Refusal, error) {
	standings, err := b.standings(ctx, payers)
	if err != nil {
		return nil, err
	}

	refused := map[clicks.Payer]clicks.Refusal{}
	for payer, standing := range standings {
		if !standing.until.IsZero() {
			refused[payer] = clicks.NewRefusal(standing.until)
		}
	}
	return refused, nil
}

type standing struct {
	until  time.Time
	warned bool
}

func (s standing) refusal() (clicks.Refusal, bool) {
	switch {
	case !s.until.IsZero():
		return clicks.NewRefusal(s.until), true
	case s.warned:
		return clicks.NewWarning(), true
	default:
		return clicks.Refusal{}, false
	}
}

func (b Book) standings(ctx context.Context, payers []clicks.Payer) (map[clicks.Payer]standing, error) {
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
	standings := make(map[clicks.Payer]standing, len(payers))
	for _, payer := range payers {
		var s standing
		for _, caller := range callersOf(payer) {
			refusal := kept[caller]
			if refusal.banned(now) && refusal.until.After(s.until) {
				s.until = refusal.until
			}
			s.warned = s.warned || refusal.warning
		}
		standings[payer] = s
	}
	return standings, nil
}

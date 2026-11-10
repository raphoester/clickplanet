package warnings

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Warning struct {
	Account        string
	At             time.Time
	AcknowledgedAt time.Time
}

func (w Warning) Pending() bool { return w.AcknowledgedAt.IsZero() }

type Store interface {
	// Warned holds each of the accounts ever warned; an account never warned is not in it.
	Warned(ctx context.Context, accounts []string) (map[string]Warning, error)
	Since(ctx context.Context, from time.Time) ([]Warning, error)
	// Keep keeps a warning unless its account was warned before, and answers whether it kept it.
	Keep(ctx context.Context, warning Warning) (bool, error)
	// Acknowledge keeps the first acknowledgement of the account's warning; an account never warned changes nothing.
	Acknowledge(ctx context.Context, account string, at time.Time) error
}

// Whoever shows the account its warning while it plays.
type Feed interface {
	Publish(warning Warning)
}

type Enforcing interface {
	Enforcing() bool
}

func NewBook(enforcing Enforcing, clock cptime.Clock, store Store, feed Feed) *Book {
	return &Book{enforcing: enforcing, clock: clock, store: store, feed: feed}
}

type Book struct {
	enforcing Enforcing
	clock     cptime.Clock
	store     Store
	feed      Feed
}

func (b *Book) Warned(ctx context.Context, accounts []string) (bool, error) {
	warned, err := b.store.Warned(ctx, accounts)
	if err != nil {
		return false, fmt.Errorf("failed to read the warnings of %d accounts: %w", len(accounts), err)
	}
	return len(warned) > 0, nil
}

// Not enforced, a warning is neither kept nor shown: kept, it would ban a ring that never saw it.
func (b *Book) Warn(ctx context.Context, accounts []string) ([]Warning, error) {
	if !b.enforcing.Enforcing() {
		return nil, nil
	}

	var warned []Warning
	for _, account := range accounts {
		warning := Warning{Account: account, At: b.clock.Now()}

		kept, err := b.store.Keep(ctx, warning)
		if err != nil {
			return warned, fmt.Errorf("failed to warn %q: %w", account, err)
		}
		if kept {
			b.feed.Publish(warning)
			warned = append(warned, warning)
		}
	}
	return warned, nil
}

func (b *Book) Acknowledge(ctx context.Context, account string) error {
	if err := b.store.Acknowledge(ctx, account, b.clock.Now()); err != nil {
		return fmt.Errorf("failed to keep the warning of %q acknowledged: %w", account, err)
	}
	return nil
}

func (b *Book) Pending(ctx context.Context, account string) (bool, error) {
	warning, warned, err := b.Warning(ctx, account)
	return warned && warning.Pending(), err
}

func (b *Book) Warning(ctx context.Context, account string) (Warning, bool, error) {
	if account == "" {
		return Warning{}, false, nil
	}

	warned, err := b.store.Warned(ctx, []string{account})
	if err != nil {
		return Warning{}, false, fmt.Errorf("failed to read the warning of %q: %w", account, err)
	}
	warning, ok := warned[account]
	return warning, ok, nil
}

func (b *Book) Since(ctx context.Context, from time.Time) ([]Warning, error) {
	warned, err := b.store.Since(ctx, from)
	if err != nil {
		return nil, fmt.Errorf("failed to read the warnings since %s: %w", from, err)
	}
	return warned, nil
}

package warnings

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Warning struct {
	Account string
	At      time.Time
}

type Store interface {
	// Warned holds each of the accounts ever warned; an account never warned is not in it.
	Warned(ctx context.Context, accounts []string) (map[string]Warning, error)
	Since(ctx context.Context, from time.Time) ([]Warning, error)
	// Keep keeps a warning unless its account was warned before, and answers whether it kept it.
	Keep(ctx context.Context, warning Warning) (bool, error)
}

// What a warning asks of the game: the account is refused until it acknowledges it.
type Teller interface {
	Tell(ctx context.Context, warning Warning) error
}

type Enforcing interface {
	Enforcing() bool
}

func NewBook(
	enforcing Enforcing,
	clock cptime.Clock,
	store Store,
	transactioner cpatomic.Transactioner,
	teller Teller,
) *Book {
	return &Book{enforcing: enforcing, clock: clock, store: store, transactioner: transactioner, teller: teller}
}

type Book struct {
	enforcing     Enforcing
	clock         cptime.Clock
	store         Store
	transactioner cpatomic.Transactioner
	teller        Teller
}

func (b *Book) Warned(ctx context.Context, accounts []string) (bool, error) {
	warned, err := b.store.Warned(ctx, accounts)
	if err != nil {
		return false, fmt.Errorf("failed to read the warnings of %d accounts: %w", len(accounts), err)
	}
	return len(warned) > 0, nil
}

// Not enforced, a warning is neither kept nor told: kept, it would ban a ring that never saw it.
func (b *Book) Warn(ctx context.Context, accounts []string) ([]Warning, error) {
	if !b.enforcing.Enforcing() {
		return nil, nil
	}

	var warned []Warning
	for _, account := range accounts {
		warning := Warning{Account: account, At: b.clock.Now()}

		var kept bool
		if err := b.transactioner.InTx(ctx, func(ctx context.Context) error {
			var err error
			if kept, err = b.store.Keep(ctx, warning); err != nil || !kept {
				return err //nolint:wrapcheck // the book names the account below.
			}
			return b.teller.Tell(ctx, warning) //nolint:wrapcheck // the book names the account below.
		}); err != nil {
			return warned, fmt.Errorf("failed to warn %q: %w", account, err)
		}

		if kept {
			warned = append(warned, warning)
		}
	}
	return warned, nil
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

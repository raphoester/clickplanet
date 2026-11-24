package backfill_addresses_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type Notes interface {
	Unaddressed(ctx context.Context) ([]notes.Note, error)
	Save(ctx context.Context, notes ...notes.Note) error
}

type Addresses interface {
	// The accounts that hold none are left out.
	Addresses(ctx context.Context, accounts []notes.AccountID) (map[notes.AccountID]cpmail.Address, error)
}

type Accounts interface {
	Linked(ctx context.Context, accounts []notes.AccountID) (*cpcolls.Set[notes.AccountID], error)
}

type Executor interface {
	Execute(ctx context.Context, dryRun bool) (notes.BackfillCounts, error)
}

const timeout = 2 * time.Minute

func New(kept Notes, addresses Addresses, accounts Accounts) *UseCase {
	return &UseCase{notes: kept, addresses: addresses, accounts: accounts}
}

type UseCase struct {
	notes     Notes
	addresses Addresses
	accounts  Accounts
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context, dryRun bool) (notes.BackfillCounts, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	unaddressed, err := u.notes.Unaddressed(ctx)
	if err != nil {
		return notes.BackfillCounts{}, fmt.Errorf("failed to read the notes with no address: %w", err)
	}
	senders := notes.AccountsOf(unaddressed)
	held, err := u.addresses.Addresses(ctx, senders)
	if err != nil {
		return notes.BackfillCounts{}, fmt.Errorf("failed to read the senders' addresses: %w", err)
	}
	linked, err := u.accounts.Linked(ctx, senders)
	if err != nil {
		return notes.BackfillCounts{}, fmt.Errorf("failed to read which senders signed in: %w", err)
	}

	filled, counts := notes.Backfilled(unaddressed, held, linked)
	if dryRun {
		return counts, nil
	}
	if err := u.notes.Save(ctx, filled...); err != nil {
		return notes.BackfillCounts{}, fmt.Errorf("failed to keep the addresses: %w", err)
	}
	return counts, nil
}

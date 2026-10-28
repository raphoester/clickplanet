package payroll

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/planetgrant"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Store interface {
	Brief(ctx context.Context, day missions.Day, account missions.AccountID) (missions.Brief, error)
	Sheet(ctx context.Context, day missions.Day, account missions.AccountID) (missions.Sheet, error)
	ChangeSheet(ctx context.Context, day missions.Day, account missions.AccountID, change func(missions.Sheet) missions.Sheet) error
}

type Accounts interface {
	Linked(ctx context.Context, account missions.AccountID) (bool, error)
}

func New(store Store, accounts Accounts, outbox cpatomic.Outbox) Payroll {
	return Payroll{store: store, accounts: accounts, outbox: outbox}
}

type Payroll struct {
	store    Store
	accounts Accounts
	outbox   cpatomic.Outbox
}

// Pay runs in a transaction: the grants are recorded with the payments, or neither is.
func (p Payroll) Pay(ctx context.Context, day missions.Day, account missions.AccountID) error {
	brief, err := p.store.Brief(ctx, day, account)
	if errors.Is(err, missions.ErrNoBrief) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read the day's missions: %w", err)
	}
	sheet, err := p.store.Sheet(ctx, day, account)
	if err != nil {
		return fmt.Errorf("failed to read the day's progress: %w", err)
	}
	if _, due := sheet.WithPayments(brief); len(due) == 0 {
		return nil
	}

	linked, err := p.accounts.Linked(ctx, account)
	if err != nil {
		return err //nolint:wrapcheck // the reader named it.
	}
	if !linked {
		return nil
	}

	var paid []missions.Payment
	if err := p.store.ChangeSheet(ctx, day, account, func(sheet missions.Sheet) missions.Sheet {
		next, payments := sheet.WithPayments(brief)
		paid = payments
		return next
	}); err != nil {
		return fmt.Errorf("failed to keep the day's payments: %w", err)
	}

	grants := make([]proto.Message, 0, len(paid))
	for _, payment := range paid {
		grants = append(grants, planetgrant.Of(account, payment))
	}
	if err := p.outbox.Record(ctx, grants...); err != nil {
		return fmt.Errorf("failed to record the day's grants: %w", err)
	}
	return nil
}

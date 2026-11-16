package payroll

import (
	"context"
	"errors"
	"fmt"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Store interface {
	Brief(ctx context.Context, day days.Day, account missions.AccountID) (missions.Brief, error)
	Sheet(ctx context.Context, day days.Day, account missions.AccountID) (missions.Sheet, error)
	ChangeSheet(ctx context.Context, day days.Day, account missions.AccountID, change func(missions.Sheet) missions.Sheet) error
	PaidInAll(ctx context.Context, account missions.AccountID) (uint64, error)
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

// Pay runs in a transaction: MissionsPaid is recorded with the payments, or neither is. A mission pays XP, which
// progression counts from MissionsDone: nothing is granted here.
func (p Payroll) Pay(ctx context.Context, day days.Day, account missions.AccountID) error {
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

	var paid []missions.MissionID
	if err := p.store.ChangeSheet(ctx, day, account, func(sheet missions.Sheet) missions.Sheet {
		next, payments := sheet.WithPayments(brief)
		paid = payments
		return next
	}); err != nil {
		return fmt.Errorf("failed to keep the day's payments: %w", err)
	}

	if len(paid) == 0 {
		return nil
	}
	inAll, err := p.store.PaidInAll(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to count the account's missions paid: %w", err)
	}

	if err := p.outbox.Record(ctx, &missionsv1.MissionsPaid{AccountId: account.String(), PaidInAll: inAll}); err != nil {
		return fmt.Errorf("failed to tell the day's payments: %w", err)
	}
	return nil
}

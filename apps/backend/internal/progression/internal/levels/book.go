package levels

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

// Bonuses is the bonuses module, told in its own words every kind an account unlocked, the whole list each time, and
// asked to pay a level reached by playing.
type Bonuses interface {
	KeepUnlocks(ctx context.Context, account accounts.AccountID, kinds []Kind, at time.Time) error
	PayLevelUp(ctx context.Context, account accounts.AccountID, level Level, unlocked []Kind) error
}

// Reaching level 1 is no level-up: a linked account starts there.
const firstLevel Level = 1

func NewBook(transactioner cpatomic.Transactioner, store Store, bonuses Bonuses) Book {
	return Book{transactioner: transactioner, store: store, bonuses: bonuses}
}

// Book keeps each account's level and tells the bonuses module what it unlocks in the transaction that writes it, so a
// level kept is never told late or lost.
type Book struct {
	transactioner cpatomic.Transactioner
	store         Store
	bonuses       Bonuses
	ladder        Ladder
}

func (b Book) Levels(ctx context.Context, accounts []accounts.AccountID) (ByAccount, error) {
	return b.store.Levels(ctx, accounts) //nolint:wrapcheck // the store named it.
}

// Raise writes the level only above the one kept, and answers the one kept before, 0 for none. A raise that writes tells
// every kind the level unlocks, even when the raise unlocks nothing new: the list is whole, and the latest wins. It pays
// one level-up for the raise, however many levels it jumps, from level 2 on.
func (b Book) Raise(ctx context.Context, account accounts.AccountID, level Level, at time.Time) (Level, error) {
	var previous Level
	err := b.transactioner.InTx(ctx, func(ctx context.Context) error {
		var err error
		previous, err = b.store.Raise(ctx, account, level, at)
		if err != nil {
			return err //nolint:wrapcheck // the store named it.
		}
		if previous >= level {
			return nil
		}
		unlocked := b.ladder.UnlockedAt(level)
		if err := b.bonuses.KeepUnlocks(ctx, account, unlocked, at); err != nil {
			return fmt.Errorf("failed to tell the bonuses what level %d unlocks: %w", level, err)
		}
		if level <= firstLevel {
			return nil
		}
		if err := b.bonuses.PayLevelUp(ctx, account, level, unlocked); err != nil {
			return fmt.Errorf("failed to ask the bonuses to pay level %d: %w", level, err)
		}
		return nil
	})
	return previous, err //nolint:wrapcheck // named inside.
}

// Reconcile writes a page's reconciliation and tells the bonuses what every account of the page unlocked, at the level
// it holds: none for a guest. It pays no level-up: a reconciliation is no play. at is when the reconciliation started,
// so a live raise after it wins at the bonuses.
func (b Book) Reconcile(ctx context.Context, reconciliation Reconciliation, at time.Time) error {
	return b.transactioner.InTx(ctx, func(ctx context.Context) error { //nolint:wrapcheck // named inside.
		if err := b.store.RaiseAll(ctx, reconciliation.Raised(), at); err != nil {
			return fmt.Errorf("failed to raise the levels: %w", err)
		}
		if err := b.store.Forget(ctx, reconciliation.Removed()...); err != nil {
			return fmt.Errorf("failed to remove the guests' levels: %w", err)
		}
		for account, level := range reconciliation.Reached() {
			if err := b.bonuses.KeepUnlocks(ctx, account, b.ladder.UnlockedAt(level), at); err != nil {
				return fmt.Errorf("failed to tell the bonuses what %s unlocked: %w", account, err)
			}
		}
		for _, guest := range reconciliation.Removed() {
			if err := b.bonuses.KeepUnlocks(ctx, guest, nil, at); err != nil {
				return fmt.Errorf("failed to tell the bonuses %s unlocked nothing: %w", guest, err)
			}
		}
		return nil
	})
}

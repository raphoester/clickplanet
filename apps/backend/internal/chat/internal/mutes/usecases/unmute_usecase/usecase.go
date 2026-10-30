package unmute_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store interface {
	MutesOf(ctx context.Context, account messages.AccountID) ([]mutes.Mute, error)
	Save(ctx context.Context, mute mutes.Mute) error
}

type Executor interface {
	Execute(ctx context.Context, account messages.AccountID) (time.Time, error)
}

const writeTimeout = 5 * time.Second

func New(transactioner cpatomic.Transactioner, store Store, clock cptime.Clock) *UseCase {
	return &UseCase{transactioner: transactioner, store: store, clock: clock}
}

type UseCase struct {
	transactioner cpatomic.Transactioner
	store         Store
	clock         cptime.Clock
}

var _ Executor = (*UseCase)(nil)

// Execute answers the end the mute that would have ended last had.
func (u *UseCase) Execute(ctx context.Context, account messages.AccountID) (time.Time, error) {
	if account == messages.NoAccount {
		return time.Time{}, mutes.ErrNoAccount
	}

	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	now := u.clock.Now()
	var until time.Time
	err := u.transactioner.InTx(ctx, func(ctx context.Context) error {
		given, err := u.store.MutesOf(ctx, account)
		if err != nil {
			return fmt.Errorf("failed to read the account's mutes: %w", err)
		}
		for _, mute := range given {
			if mute.Ended(now) {
				continue
			}
			if err := u.store.Save(ctx, mute.Lifted(now)); err != nil {
				return fmt.Errorf("failed to lift a mute: %w", err)
			}
			until = later(until, mute.Until())
		}
		return nil
	})
	if err != nil {
		return time.Time{}, fmt.Errorf("cannot unmute: %w", err)
	}
	if until.IsZero() {
		return time.Time{}, fmt.Errorf("%w: %s", mutes.ErrNotMuted, account)
	}
	return until, nil
}

func later(a time.Time, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

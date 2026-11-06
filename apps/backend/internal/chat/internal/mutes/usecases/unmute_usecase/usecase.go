package unmute_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store interface {
	MutesOf(ctx context.Context, account messages.AccountID, kind mutes.Kind) ([]mutes.Mute, error)
	Save(ctx context.Context, mute mutes.Mute) error
}

type Recorder interface {
	Record(ctx context.Context, action moderation.Action) error
}

type Executor interface {
	Execute(ctx context.Context, in In) (time.Time, error)
}

type In struct {
	Account messages.AccountID
	Kind    mutes.Kind
	Reason  string
	By      moderation.Operator
}

const writeTimeout = 5 * time.Second

func New(transactioner cpatomic.Transactioner, store Store, history Recorder, clock cptime.Clock) *UseCase {
	return &UseCase{transactioner: transactioner, store: store, history: history, clock: clock}
}

type UseCase struct {
	transactioner cpatomic.Transactioner
	store         Store
	history       Recorder
	clock         cptime.Clock
}

var _ Executor = (*UseCase)(nil)

// Execute answers the end the mute that would have ended last had. A mute of another kind stays. An unmute that
// lifts nothing is NotFound, and no action.
func (u *UseCase) Execute(ctx context.Context, in In) (time.Time, error) {
	account, kind := in.Account, in.Kind
	if account == messages.NoAccount {
		return time.Time{}, mutes.ErrNoAccount
	}
	reason, err := moderation.ReasonOf(in.Reason)
	if err != nil {
		return time.Time{}, fmt.Errorf("cannot unmute: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	now := u.clock.Now()
	var until time.Time
	err = u.transactioner.InTx(ctx, func(ctx context.Context) error {
		given, err := u.store.MutesOf(ctx, account, kind)
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
		if until.IsZero() {
			return nil
		}
		if err := u.history.Record(ctx, moderation.NewUnmute(account, kind, now, reason, in.By)); err != nil {
			return fmt.Errorf("failed to record the unmute in the moderation history: %w", err)
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

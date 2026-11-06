package mute_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Saver interface {
	Save(ctx context.Context, mute mutes.Mute) error
}

type Recorder interface {
	Record(ctx context.Context, action moderation.Action) error
}

type Addresses interface {
	LatestAddress(ctx context.Context, account messages.AccountID) (string, error)
}

type Executor interface {
	Execute(ctx context.Context, in In) (mutes.Mute, error)
}

type In struct {
	Account  messages.AccountID
	Kind     mutes.Kind
	Duration time.Duration
	Reason   string
	By       moderation.Operator
}

const writeTimeout = 5 * time.Second

func New(
	transactioner cpatomic.Transactioner, saver Saver, history Recorder, addresses Addresses, ids mutes.IDProvider,
	clock cptime.Clock,
) *UseCase {
	return &UseCase{
		transactioner: transactioner, saver: saver, history: history, addresses: addresses, ids: ids, clock: clock,
	}
}

type UseCase struct {
	transactioner cpatomic.Transactioner
	saver         Saver
	history       Recorder
	addresses     Addresses
	ids           mutes.IDProvider
	clock         cptime.Clock
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context, in In) (mutes.Mute, error) {
	if in.Account == messages.NoAccount {
		return mutes.Mute{}, mutes.ErrNoAccount
	}
	duration, err := mutes.DurationOf(in.Duration)
	if err != nil {
		return mutes.Mute{}, fmt.Errorf("cannot mute: %w", err)
	}
	reason, err := moderation.ReasonOf(in.Reason)
	if err != nil {
		return mutes.Mute{}, fmt.Errorf("cannot mute: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	ip, err := u.addresses.LatestAddress(ctx, in.Account)
	if err != nil && !errors.Is(err, messages.ErrNoMessage) {
		return mutes.Mute{}, fmt.Errorf("failed to read where the account posts from: %w", err)
	}

	id, err := u.ids.NewID()
	if err != nil {
		return mutes.Mute{}, fmt.Errorf("failed to draw a mute id: %w", err)
	}

	mute := mutes.NewMute(id, mutes.NewCaller(in.Account, ip), in.Kind, u.clock.Now(), duration)
	err = u.transactioner.InTx(ctx, func(ctx context.Context) error {
		if err := u.saver.Save(ctx, mute); err != nil {
			return fmt.Errorf("failed to keep the mute: %w", err)
		}
		if err := u.history.Record(ctx, moderation.NewMute(mute, reason, in.By)); err != nil {
			return fmt.Errorf("failed to record the mute in the moderation history: %w", err)
		}
		return nil
	})
	if err != nil {
		return mutes.Mute{}, fmt.Errorf("cannot mute: %w", err)
	}
	return mute, nil
}

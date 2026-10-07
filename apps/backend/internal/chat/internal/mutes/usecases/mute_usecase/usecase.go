package mute_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Saver interface {
	Save(ctx context.Context, mute mutes.Mute) error
}

type Addresses interface {
	LatestAddress(ctx context.Context, account messages.AccountID) (string, error)
}

type Executor interface {
	Execute(ctx context.Context, in In) (mutes.Mute, error)
}

type In struct {
	Account  messages.AccountID
	Duration time.Duration
}

const writeTimeout = 5 * time.Second

func New(saver Saver, addresses Addresses, ids mutes.IDProvider, clock cptime.Clock) *UseCase {
	return &UseCase{saver: saver, addresses: addresses, ids: ids, clock: clock}
}

type UseCase struct {
	saver     Saver
	addresses Addresses
	ids       mutes.IDProvider
	clock     cptime.Clock
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

	mute := mutes.NewMute(id, mutes.NewCaller(in.Account, ip), u.clock.Now(), duration)
	if err := u.saver.Save(ctx, mute); err != nil {
		return mutes.Mute{}, fmt.Errorf("failed to keep the mute: %w", err)
	}
	return mute, nil
}

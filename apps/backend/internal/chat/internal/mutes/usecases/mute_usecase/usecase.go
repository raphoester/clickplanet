package mute_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_usecase"
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

type Authors interface {
	Author(ctx context.Context, account messages.AccountID) (messages.Author, error)
}

type Announcer interface {
	Execute(ctx context.Context, in announce_usecase.In) error
}

type Executor interface {
	Execute(ctx context.Context, in In) (Out, error)
}

type In struct {
	Account  messages.AccountID
	Duration time.Duration
}

type Out struct {
	Name string
	Mute mutes.Mute
}

const writeTimeout = 5 * time.Second

func New(saver Saver, addresses Addresses, authors Authors, announcer Announcer, clock cptime.Clock) *UseCase {
	return &UseCase{saver: saver, addresses: addresses, authors: authors, announcer: announcer, clock: clock}
}

type UseCase struct {
	saver     Saver
	addresses Addresses
	authors   Authors
	announcer Announcer
	clock     cptime.Clock
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	if in.Account == messages.NoAccount {
		return Out{}, mutes.ErrNoAccount
	}
	duration, err := mutes.DurationOf(in.Duration)
	if err != nil {
		return Out{}, fmt.Errorf("cannot mute: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	author, err := u.authors.Author(ctx, in.Account)
	if err != nil {
		return Out{}, fmt.Errorf("failed to name the account to mute: %w", err)
	}

	ip, err := u.addresses.LatestAddress(ctx, in.Account)
	if err != nil && !errors.Is(err, messages.ErrNoMessage) {
		return Out{}, fmt.Errorf("failed to read where the account posts from: %w", err)
	}

	now := u.clock.Now()
	mute := mutes.NewMute(mutes.MuteID(uuid.New()), mutes.NewCaller(in.Account, ip), now, duration)
	if err := u.saver.Save(ctx, mute); err != nil {
		return Out{}, fmt.Errorf("failed to keep the mute: %w", err)
	}

	payload, err := announcements.MutedOf(author.Name(), duration).Payload()
	if err != nil {
		return Out{}, fmt.Errorf("the mute is kept but cannot be announced: %w", err)
	}
	if err := u.announcer.Execute(ctx, announce_usecase.In{Kind: announcements.KindMute, At: now, Payload: payload}); err != nil {
		return Out{}, fmt.Errorf("the mute is kept but was not announced: %w", err)
	}

	return Out{Name: author.Name(), Mute: mute}, nil
}

package announce_mute_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type Authors interface {
	Author(ctx context.Context, account messages.AccountID) (messages.Author, error)
}

type Announcer interface {
	Execute(ctx context.Context, in announce_usecase.In) error
}

type In struct {
	Account  messages.AccountID
	At       time.Time
	Duration time.Duration
}

func New(authors Authors, announcer Announcer) *UseCase {
	return &UseCase{authors: authors, announcer: announcer}
}

type UseCase struct {
	authors   Authors
	announcer Announcer
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	author, err := u.authors.Author(ctx, in.Account)
	if err != nil {
		return fmt.Errorf("failed to name the muted account: %w", err)
	}

	payload, err := announcements.MutedOf(author.Name(), in.Duration).Payload()
	if err != nil {
		return fmt.Errorf("cannot announce the mute: %w", err)
	}

	return u.announcer.Execute(ctx, announce_usecase.In{Kind: announcements.KindMute, At: in.At, Payload: payload})
}

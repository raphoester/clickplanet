package announce_mute_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
)

type Authors interface {
	Author(ctx context.Context, account messages.AccountID) (messages.Author, error)
}

type Announcer interface {
	Execute(ctx context.Context, announcement announcements.Announcement) error
}

type In struct {
	Mute     mutes.MuteID
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

	announcement, err := announcements.MutedOf(in.Mute, author.Name(), in.Duration).Announcement(in.At)
	if err != nil {
		return fmt.Errorf("cannot announce the mute: %w", err)
	}

	return u.announcer.Execute(ctx, announcement)
}

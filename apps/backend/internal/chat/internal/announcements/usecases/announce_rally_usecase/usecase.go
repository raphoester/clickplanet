package announce_rally_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type Authors interface {
	Author(ctx context.Context, account messages.AccountID) (messages.Author, error)
}

type Announcer interface {
	Execute(ctx context.Context, announcement announcements.Announcement) error
}

type In struct {
	Rally    announcements.RallyID
	Account  messages.AccountID
	Country  string
	Ground   string
	Landmass uint32
	At       time.Time
}

func New(authors Authors, announcer Announcer) *UseCase {
	return &UseCase{authors: authors, announcer: announcer}
}

type UseCase struct {
	authors   Authors
	announcer Announcer
}

// Execute tells the flag's room of a rally under the name its caller has now.
func (u *UseCase) Execute(ctx context.Context, in In) error {
	author, err := u.authors.Author(ctx, in.Account)
	if err != nil {
		return fmt.Errorf("failed to name the rally's caller: %w", err)
	}

	announcement, err := announcements.RallyOf(in.Rally, author.Name(), in.Country, in.Ground, in.Landmass).Announcement(in.At)
	if err != nil {
		return fmt.Errorf("cannot announce the rally: %w", err)
	}

	return u.announcer.Execute(ctx, announcement) //nolint:wrapcheck // the announcer named it.
}

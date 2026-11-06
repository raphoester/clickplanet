package forget_link_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
)

type Store interface {
	LinkOf(ctx context.Context, account links.AccountID) (links.Link, error)
	SaveLinks(ctx context.Context, links ...links.Link) error
}

type Members interface {
	Rename(ctx context.Context, user discord.UserID, nick string) error
}

const timeout = 10 * time.Second

func New(store Store, members Members) *UseCase {
	return &UseCase{store: store, members: members}
}

type UseCase struct {
	store   Store
	members Members
}

// The Discord user stays in the server with its roles: that is its own Discord account, not the game's data. The
// nickname the bot gave it is the game's: the username player forgot, so it is taken off first, and Discord shows the
// user's own name again. A user who left the server, or one the bot may no longer rename, is forgotten all the same;
// Discord busy or down is tried again, the link kept until then.
func (u *UseCase) Execute(ctx context.Context, account links.AccountID) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	link, err := u.store.LinkOf(ctx, account)
	if errors.Is(err, links.ErrNoLink) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read the link: %w", err)
	}
	if nickname, renamed := link.Rename(""); renamed {
		err := u.members.Rename(ctx, nickname.User, "")
		if err != nil && !errors.Is(err, discord.ErrNotMember) && !errors.Is(err, discord.ErrRefused) {
			return err //nolint:wrapcheck // the client named it.
		}
	}
	link.Forget()
	if err := u.store.SaveLinks(ctx, link); err != nil {
		return fmt.Errorf("failed to forget the link: %w", err)
	}
	return nil
}

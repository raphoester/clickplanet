package links

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
)

// The bot's hand on a member's nickname in the server.
type Renamer interface {
	Rename(ctx context.Context, user discord.UserID, nick string) error
}

func NewNamer(store Store, names Names, members Renamer) Namer {
	return Namer{store: store, names: names, members: members}
}

// Gives each linked Discord user its account's username as its nickname in the server: the name players see in the
// game is the one they see on Discord.
type Namer struct {
	store   Store
	names   Names
	members Renamer
}

// Asks player the account's name at each call, so a call made late, twice, or after a later one, leaves the user with
// the name the account holds now. No name takes the nickname off: a guest, or an account its player asked to delete.
// True when the nickname changed; false for an account with no link, or a user wearing its name already. A user who
// left the server is forgotten, as when its flag moves. discord.ErrRefused: the bot may not rename this member (the
// server's owner, or a member whose highest role is above the bot's), or Discord took no such nickname.
func (n Namer) Name(ctx context.Context, account AccountID) (bool, error) {
	link, err := n.store.LinkOf(ctx, account)
	if errors.Is(err, ErrNoLink) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to read the link: %w", err)
	}
	name, err := n.names.NameOf(ctx, account)
	if err != nil {
		return false, err //nolint:wrapcheck // the adapter named it.
	}
	nickname, renamed := link.Rename(name)
	if !renamed {
		return false, nil
	}

	err = n.members.Rename(ctx, nickname.User, string(nickname.Name))
	if errors.Is(err, discord.ErrNotMember) {
		link.Forget()
		if err := n.store.SaveLinks(ctx, link); err != nil {
			return false, fmt.Errorf("failed to forget the link of a user who left the server: %w", err)
		}
		return false, nil
	}
	if err != nil {
		return false, err //nolint:wrapcheck // the client named it.
	}
	if err := n.store.SaveName(ctx, link); err != nil {
		return false, fmt.Errorf("failed to keep the nickname given: %w", err)
	}
	return true, nil
}

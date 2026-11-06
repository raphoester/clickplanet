package unlink_player_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles"
)

type Store interface {
	LinkOf(ctx context.Context, account links.AccountID) (links.Link, error)
	SaveLinks(ctx context.Context, links ...links.Link) error
}

type Roles interface {
	IDOf(ctx context.Context, role roles.Role) (discord.RoleID, error)
}

type Members interface {
	Revoke(ctx context.Context, user discord.UserID, role discord.RoleID) error
	Rename(ctx context.Context, user discord.UserID, nick string) error
}

const timeout = 10 * time.Second

func New(store Store, book Roles, members Members) *UseCase {
	return &UseCase{store: store, roles: book, members: members}
}

type UseCase struct {
	store   Store
	roles   Roles
	members Members
}

// False when the account was not linked. The user stays in the server, without the flag's role, which no flag the
// account pledges would move any more, and without the nickname the bot gave it, which no rename would follow; a user
// who left the server already has neither to lose.
func (u *UseCase) Execute(ctx context.Context, account links.AccountID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	link, err := u.store.LinkOf(ctx, account)
	if errors.Is(err, links.ErrNoLink) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to read the link: %w", err)
	}
	if err := u.unname(ctx, &link); err != nil {
		return false, err
	}
	if err := u.revoke(ctx, link.Forget()); err != nil {
		return false, err
	}
	if err := u.store.SaveLinks(ctx, link); err != nil {
		return false, fmt.Errorf("failed to forget the link: %w", err)
	}
	return true, nil
}

func (u *UseCase) unname(ctx context.Context, link *links.Link) error {
	nickname, renamed := link.Rename("")
	if !renamed {
		return nil
	}
	err := u.members.Rename(ctx, nickname.User, "")
	if err != nil && !errors.Is(err, discord.ErrNotMember) {
		return err //nolint:wrapcheck // the client named it.
	}
	return nil
}

func (u *UseCase) revoke(ctx context.Context, move links.Move) error {
	if move.From == "" {
		return nil
	}
	role, err := u.roles.IDOf(ctx, roles.Nation(move.From))
	if errors.Is(err, roles.ErrNoRole) {
		return nil
	}
	if err != nil {
		return err //nolint:wrapcheck // the book named it.
	}
	err = u.members.Revoke(ctx, move.User, role)
	if err != nil && !errors.Is(err, discord.ErrNotMember) {
		return err //nolint:wrapcheck // the client named it.
	}
	return nil
}

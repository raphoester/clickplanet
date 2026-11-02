package follow_flag_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles"
)

type Store interface {
	LinkOf(ctx context.Context, account links.AccountID) (links.Link, error)
	SaveLinks(ctx context.Context, links ...links.Link) error
}

type Roles interface {
	IDOf(ctx context.Context, role roles.Role) (discord.RoleID, error)
	Ensure(ctx context.Context, role roles.Role) (discord.RoleID, error)
}

type Members interface {
	Grant(ctx context.Context, user discord.UserID, role discord.RoleID) error
	Revoke(ctx context.Context, user discord.UserID, role discord.RoleID) error
}

type In struct {
	Account links.AccountID
	Flag    nations.Flag
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

// A user who left the server is forgotten: it asks to be pinged no more.
func (u *UseCase) Execute(ctx context.Context, in In) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	link, err := u.store.LinkOf(ctx, in.Account)
	if errors.Is(err, links.ErrNoLink) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read the link: %w", err)
	}
	move, moved := link.Follow(in.Flag)
	if !moved {
		return nil
	}

	err = u.swap(ctx, move)
	if errors.Is(err, discord.ErrNotMember) {
		link.Forget()
	} else if err != nil {
		return err
	}
	if err := u.store.SaveLinks(ctx, link); err != nil {
		return fmt.Errorf("failed to keep the link: %w", err)
	}
	return nil
}

func (u *UseCase) swap(ctx context.Context, move links.Move) error {
	granted, err := u.roles.Ensure(ctx, roles.Nation(move.To))
	if err != nil {
		return err //nolint:wrapcheck // the book named it.
	}
	if err := u.members.Grant(ctx, move.User, granted); err != nil {
		return err //nolint:wrapcheck // the client named it.
	}
	if move.From == "" {
		return nil
	}
	revoked, err := u.roles.IDOf(ctx, roles.Nation(move.From))
	if errors.Is(err, roles.ErrNoRole) {
		return nil
	}
	if err != nil {
		return err //nolint:wrapcheck // the book named it.
	}
	return u.members.Revoke(ctx, move.User, revoked) //nolint:wrapcheck // the client named it.
}

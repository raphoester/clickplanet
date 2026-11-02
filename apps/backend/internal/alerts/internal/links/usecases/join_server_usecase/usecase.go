package join_server_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store interface {
	LinkOf(ctx context.Context, account links.AccountID) (links.Link, error)
	LinkOfUser(ctx context.Context, user discord.UserID) (links.Link, error)
	SaveLinks(ctx context.Context, links ...links.Link) error
}

type Discord interface {
	Join(ctx context.Context, user discord.UserID, token discord.AccessToken, roles []discord.RoleID) error
}

type Allegiances interface {
	FlagOf(ctx context.Context, account links.AccountID) (nations.Flag, error)
}

type Roles interface {
	Ensure(ctx context.Context, role roles.Role) (discord.RoleID, error)
}

// What a Discord sign-in proved: the account it signed in to, the Discord user, and a token that may join the server.
type In struct {
	Account links.AccountID
	User    discord.UserID
	Token   discord.AccessToken
}

const timeout = 15 * time.Second

func New(clock cptime.Clock, store Store, server Discord, allegiances Allegiances, book Roles) *UseCase {
	return &UseCase{clock: clock, store: store, discord: server, allegiances: allegiances, roles: book}
}

type UseCase struct {
	clock       cptime.Clock
	store       Store
	discord     Discord
	allegiances Allegiances
	roles       Roles
}

// An account linked to this user already changes nothing: a user who left the server since is not added back by
// every sign-in. A user linked from another account moves here.
func (u *UseCase) Execute(ctx context.Context, in In) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	kept, err := u.store.LinkOf(ctx, in.Account)
	switch {
	case errors.Is(err, links.ErrNoLink):
	case err != nil:
		return fmt.Errorf("failed to read the account's link: %w", err)
	case kept.Joins(in.Account, in.User):
		return nil
	}

	flag, err := u.allegiances.FlagOf(ctx, in.Account)
	if err != nil {
		return err //nolint:wrapcheck // the adapter named it.
	}
	var granted []discord.RoleID
	if flag != "" {
		role, err := u.roles.Ensure(ctx, roles.Nation(flag))
		if err != nil {
			return err //nolint:wrapcheck // the book named it.
		}
		granted = append(granted, role)
	}
	if err := u.discord.Join(ctx, in.User, in.Token, granted); err != nil {
		return err //nolint:wrapcheck // the client named it.
	}

	saved := []links.Link{links.NewLink(in.Account, in.User, flag, u.clock.Now())}
	earlier, err := u.store.LinkOfUser(ctx, in.User)
	switch {
	case errors.Is(err, links.ErrNoLink):
	case err != nil:
		return fmt.Errorf("failed to read who the user linked before: %w", err)
	case !earlier.BelongsTo(in.Account):
		earlier.Forget()
		saved = append(saved, earlier)
	}
	if err := u.store.SaveLinks(ctx, saved...); err != nil {
		return fmt.Errorf("failed to keep the link: %w", err)
	}
	return nil
}

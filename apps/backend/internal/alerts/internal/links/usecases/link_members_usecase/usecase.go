package link_members_usecase

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

// Who signed in with Discord, from auth.
type Accounts interface {
	DiscordAccounts(ctx context.Context, after links.AccountID, limit int) ([]links.DiscordAccount, error)
}

type Members interface {
	IsMember(ctx context.Context, user discord.UserID) (bool, error)
	Grant(ctx context.Context, user discord.UserID, role discord.RoleID) error
}

type Allegiances interface {
	FlagOf(ctx context.Context, account links.AccountID) (nations.Flag, error)
}

type Roles interface {
	Ensure(ctx context.Context, role roles.Role) (discord.RoleID, error)
}

type Executor interface {
	Execute(ctx context.Context) (Out, error)
}

// The accounts asked of auth at once.
const PageSize = 100

// One account's checks and grant: a 429 waited out twice fits.
const accountTimeout = 30 * time.Second

// An account that could not be linked, and why. A run again tries it again.
type Failure struct {
	Account links.AccountID
	User    discord.UserID
	Err     error
}

type Out struct {
	// In the server, and linked now with its flag's role.
	Linked int
	// Not in the server: only the user's own Discord sign-in can add it.
	Outside int
	Failed  []Failure
}

func New(clock cptime.Clock, store Store, accounts Accounts, members Members, allegiances Allegiances, book Roles) *UseCase {
	return &UseCase{clock: clock, store: store, accounts: accounts, members: members, allegiances: allegiances, roles: book}
}

type UseCase struct {
	clock       cptime.Clock
	store       Store
	accounts    Accounts
	members     Members
	allegiances Allegiances
	roles       Roles
}

var _ Executor = (*UseCase)(nil)

// Links each account signed in with Discord whose user is in the server already and that has no link: it joined by
// the invite, or signed in before the sign-in asked to join. An account linked already, and a user linked from
// another account, are left as they are, so a second run links nobody twice. One account that fails does not stop the
// others; a run cut short, or auth that does not answer, stops it.
func (u *UseCase) Execute(ctx context.Context) (Out, error) {
	var (
		out   Out
		after links.AccountID
	)
	for {
		page, err := u.accounts.DiscordAccounts(ctx, after, PageSize)
		if err != nil {
			return out, err //nolint:wrapcheck // the adapter named it.
		}
		for _, signed := range page {
			if err := ctx.Err(); err != nil {
				return out, fmt.Errorf("the run was cut short: %w", err)
			}
			switch made, err := u.link(ctx, signed); {
			case err != nil:
				out.Failed = append(out.Failed, Failure{Account: signed.Account, User: signed.User, Err: err})
			case made == linked:
				out.Linked++
			case made == outside:
				out.Outside++
			}
		}
		if len(page) < PageSize {
			return out, nil
		}
		after = page[len(page)-1].Account
	}
}

type outcome int

const (
	kept outcome = iota
	linked
	outside
)

func (u *UseCase) link(ctx context.Context, signed links.DiscordAccount) (outcome, error) {
	ctx, cancel := context.WithTimeout(ctx, accountTimeout)
	defer cancel()

	if done, err := u.linkedAlready(ctx, signed); done || err != nil {
		return kept, err
	}

	member, err := u.members.IsMember(ctx, signed.User)
	if err != nil {
		return kept, err //nolint:wrapcheck // the client named it.
	}
	if !member {
		return outside, nil
	}

	flag, err := u.allegiances.FlagOf(ctx, signed.Account)
	if err != nil {
		return kept, err //nolint:wrapcheck // the adapter named it.
	}
	if flag != "" {
		role, err := u.roles.Ensure(ctx, roles.Nation(flag))
		if err != nil {
			return kept, err //nolint:wrapcheck // the book named it.
		}
		err = u.members.Grant(ctx, signed.User, role)
		if errors.Is(err, discord.ErrNotMember) {
			return outside, nil
		}
		if err != nil {
			return kept, err //nolint:wrapcheck // the client named it.
		}
	}

	if err := u.store.SaveLinks(ctx, links.NewLink(signed.Account, signed.User, flag, u.clock.Now())); err != nil {
		return kept, fmt.Errorf("failed to keep the link: %w", err)
	}
	return linked, nil
}

// The account has a link, or its user is another account's: a run never moves one.
func (u *UseCase) linkedAlready(ctx context.Context, signed links.DiscordAccount) (bool, error) {
	_, err := u.store.LinkOf(ctx, signed.Account)
	switch {
	case err == nil:
		return true, nil
	case !errors.Is(err, links.ErrNoLink):
		return false, fmt.Errorf("failed to read the account's link: %w", err)
	}
	_, err = u.store.LinkOfUser(ctx, signed.User)
	switch {
	case err == nil:
		return true, nil
	case !errors.Is(err, links.ErrNoLink):
		return false, fmt.Errorf("failed to read who the user is linked to: %w", err)
	}
	return false, nil
}

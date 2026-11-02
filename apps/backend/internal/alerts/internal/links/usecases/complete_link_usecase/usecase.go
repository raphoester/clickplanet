package complete_link_usecase

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
	FlowOf(ctx context.Context, account links.AccountID) (links.Flow, error)
	SaveFlow(ctx context.Context, flow links.Flow) error
	LinkOfUser(ctx context.Context, user discord.UserID) (links.Link, error)
	SaveLinks(ctx context.Context, links ...links.Link) error
}

type Discord interface {
	Exchange(ctx context.Context, code string, verifier string) (discord.AccessToken, error)
	UserOf(ctx context.Context, token discord.AccessToken) (discord.UserID, error)
	Join(ctx context.Context, user discord.UserID, token discord.AccessToken, roles []discord.RoleID) error
}

type Allegiances interface {
	FlagOf(ctx context.Context, account links.AccountID) (nations.Flag, error)
}

type Roles interface {
	Ensure(ctx context.Context, role roles.Role) (discord.RoleID, error)
}

type In struct {
	Account links.AccountID
	Code    string
	State   string
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

func (u *UseCase) Execute(ctx context.Context, in In) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	flow, err := u.store.FlowOf(ctx, in.Account)
	if errors.Is(err, links.ErrNoFlow) {
		return links.ErrFlowInvalid
	}
	if err != nil {
		return fmt.Errorf("failed to read the link flow: %w", err)
	}
	now := u.clock.Now()
	verifier, err := flow.Spend(in.State, now)
	if err != nil {
		return err //nolint:wrapcheck // the flow named it.
	}
	if err := u.store.SaveFlow(ctx, flow); err != nil {
		return fmt.Errorf("failed to spend the link flow: %w", err)
	}

	token, err := u.discord.Exchange(ctx, in.Code, verifier)
	if err != nil {
		return err //nolint:wrapcheck // the client named it.
	}
	user, err := u.discord.UserOf(ctx, token)
	if err != nil {
		return err //nolint:wrapcheck // the client named it.
	}
	flag, err := u.allegiances.FlagOf(ctx, in.Account)
	if err != nil {
		return err //nolint:wrapcheck // the adapter named it.
	}
	granted, err := u.rolesOf(ctx, flag)
	if err != nil {
		return err
	}
	if err := u.discord.Join(ctx, user, token, granted); err != nil {
		return err //nolint:wrapcheck // the client named it.
	}

	saved := []links.Link{links.NewLink(in.Account, user, flag, now)}
	earlier, err := u.store.LinkOfUser(ctx, user)
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

func (u *UseCase) rolesOf(ctx context.Context, flag nations.Flag) ([]discord.RoleID, error) {
	wanted := []roles.Role{roles.Battles()}
	if flag != "" {
		wanted = append(wanted, roles.Nation(flag))
	}
	granted := make([]discord.RoleID, 0, len(wanted))
	for _, role := range wanted {
		id, err := u.roles.Ensure(ctx, role)
		if err != nil {
			return nil, err //nolint:wrapcheck // the book named it.
		}
		granted = append(granted, id)
	}
	return granted, nil
}

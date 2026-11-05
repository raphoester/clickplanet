package discord_join_granted_subscriber

import (
	"context"
	"errors"
	"log/slog"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/join_server_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type Grants interface {
	Grant(ctx context.Context, id links.GrantID) (links.Grant, error)
}

type UseCase interface {
	Execute(ctx context.Context, in join_server_usecase.In) error
}

func New(grants Grants, useCase UseCase, logger *slog.Logger) Subscriber {
	return Subscriber{grants: grants, useCase: useCase, logger: logger}
}

type Subscriber struct {
	grants  Grants
	useCase UseCase
	logger  *slog.Logger
}

var _ cpredis.Handler[*authv1.DiscordJoinGranted] = Subscriber{}

// The token is auth's, asked for by the grant's id: it is never in the event. A grant auth keeps no more, or a token
// Discord refused, is logged and not tried again: the player is signed in, not linked, and the game offers to join
// again. Anything else (Discord busy or down, auth unreachable) is an error, tried again while the grant lives.
func (s Subscriber) Handle(ctx context.Context, event *authv1.DiscordJoinGranted) error {
	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	grant, err := s.grants.Grant(ctx, links.GrantID(event.GetGrantId()))
	if errors.Is(err, links.ErrNoGrant) {
		s.logged(ctx, "a Discord sign-in's grant was gone before it joined the server", event, err)
		return nil
	}
	if err != nil {
		return err //nolint:wrapcheck // the adapter named it.
	}

	err = s.useCase.Execute(ctx, join_server_usecase.In{Account: grant.Account, User: grant.User, Token: grant.Token})
	if errors.Is(err, discord.ErrRefused) {
		s.logged(ctx, "a Discord sign-in did not join the server", event, err)
		return nil
	}
	return err //nolint:wrapcheck // the use case named it.
}

func (s Subscriber) logged(ctx context.Context, message string, event *authv1.DiscordJoinGranted, err error) {
	s.logger.WarnContext(ctx, message,
		slog.String("grant", event.GetGrantId()),
		slog.String("account", event.GetAccountId()),
		slog.String("discord_user", event.GetDiscordUserId()),
		slog.Any("error", err),
	)
}

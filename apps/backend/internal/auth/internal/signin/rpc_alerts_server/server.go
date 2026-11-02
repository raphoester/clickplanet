package rpc_alerts_server

import (
	"context"
	"log/slog"
	"time"

	"connectrpc.com/connect"

	alertsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/alerts/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
)

type Alerts interface {
	JoinServer(
		ctx context.Context,
		req *connect.Request[alertsv1.JoinServerRequest],
	) (*connect.Response[alertsv1.JoinServerResponse], error)
}

// A few calls to Discord: the role, then the member. Well under the page's 10 s for CompleteSignIn, which carries the
// session cookie: a slow Discord costs the join, never the sign-in.
const askTimeout = 5 * time.Second

func New(alerts Alerts, logger *slog.Logger) Server {
	return Server{alerts: alerts, logger: logger}
}

// Asks the alerts module, which owns the game's Discord server. Its failures are logged here, since a sign-in never
// fails for them.
type Server struct {
	alerts Alerts
	logger *slog.Logger
}

var _ signin.Server = Server{}

func (s Server) Join(ctx context.Context, account accounts.AccountID, user string, token string) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	_, err := s.alerts.JoinServer(ctx, connect.NewRequest(&alertsv1.JoinServerRequest{
		AccountId: account.String(), DiscordUserId: user, AccessToken: token,
	}))
	if err == nil {
		return
	}
	// The alerts module is off: there is no server to join.
	if code := connect.CodeOf(err); code == connect.CodeUnimplemented || code == connect.CodeNotFound {
		return
	}
	s.logger.WarnContext(ctx, "a Discord sign-in did not join the server",
		slog.String("account", account.String()), slog.String("discord_user", user), slog.Any("error", err))
}

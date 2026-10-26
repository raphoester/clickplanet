package e2e_test

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1/antibotv1connect"
	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type watchedGame struct {
	baseURL    string
	consoleURL string
	verifier   *cpsession.Verifier
}

func startWatchedGame(t *testing.T) watchedGame {
	t.Helper()

	postgres := cppg.StartTestServer(t)
	redis := cpredis.StartTestServer(t).Config()
	secret, public := cpsession.TestKeyPair()
	listeners := []net.Listener{listen(t), listen(t), listen(t)}
	server := cpbootstrap.ServerConfig{
		BindAddress:         listeners[0].Addr().String(),
		InternalBindAddress: listeners[1].Addr().String(),
		ConsoleBindAddress:  listeners[2].Addr().String(),
	}

	authConfig := auth.Config{
		SignerConfig: cpsession.SignerConfig{Enabled: true, Secret: secret, TTL: time.Hour},
		Database:     postgres.ConfigFor("auth"),
		Redis:        redis,
	}
	authConfig.RateLimiter.PerSecond = 100
	authConfig.RateLimiter.Burst = 100

	planetConfig := planet.Config{Database: postgres.ConfigFor("planet"), Redis: redis}
	planetConfig.GameMap.MaxIndex = mapTiles
	planetConfig.Auth = cpsession.VerifierConfig{Enabled: true, Enforce: true}
	planetConfig.RateLimiter.PerSecond = 100
	planetConfig.RateLimiter.Burst = 100

	antibotConfig := antibot.Config{Enabled: true, Database: postgres.ConfigFor("antibot"), Redis: redis}
	antibotConfig.Bans.Enforce = true
	antibotConfig.Sequencer.Enabled = true

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- cpbootstrap.RunOn(ctx, cpbootstrap.Options{
			Server:  server,
			Logger:  slog.New(slog.DiscardHandler),
			Modules: []cpbootstrap.Module{auth.NewModule(authConfig), planet.NewModule(planetConfig), antibot.NewModule(antibotConfig)},
		}, listeners[0], listeners[1], listeners[2])
	}()
	t.Cleanup(func() {
		cancel()
		assert.NoError(t, <-done)
	})

	waitUntilServed(t, server.BindAddress)

	verifier, err := cpsession.NewVerifier(public)
	require.NoError(t, err)

	return watchedGame{baseURL: "http://" + server.BindAddress, consoleURL: "http://" + server.ConsoleBindAddress, verifier: verifier}
}

func (g watchedGame) guest(t *testing.T) (string, string) {
	t.Helper()

	req := connect.NewRequest(&authv1.CreateSessionRequest{AttestationToken: "unused"})
	req.Header().Set("X-Real-IP", callerIP)
	res, err := authv1connect.NewAuthServiceClient(http.DefaultClient, g.baseURL).CreateSession(t.Context(), req)
	require.NoError(t, err)

	claims, err := g.verifier.Verify(res.Msg.GetToken(), callerIP, time.Now())
	require.NoError(t, err)
	return res.Msg.GetToken(), claims.Account.String()
}

func (g watchedGame) click(t *testing.T, token string) error {
	t.Helper()

	clicks := planetv1connect.NewClickServiceClient(http.DefaultClient, g.baseURL)
	pledge := connect.NewRequest(&planetv1.PledgeAllegianceRequest{CountryId: "fr"})
	pledge.Header().Set("X-Real-IP", callerIP)
	pledge.Header().Set(cpconnect.SessionHeader, token)
	if _, err := clicks.PledgeAllegiance(t.Context(), pledge); err != nil {
		return err //nolint:wrapcheck // the test reads the code.
	}

	req := connect.NewRequest(&planetv1.ClickRequest{TileId: 1, CountryId: "fr"})
	req.Header().Set("X-Real-IP", callerIP)
	req.Header().Set(cpconnect.SessionHeader, token)
	_, err := clicks.Click(t.Context(), req)
	return err //nolint:wrapcheck // the test reads the code.
}

func TestABanReachesPlanetAndTheCallerIsRefused(t *testing.T) {
	game := startWatchedGame(t)
	token, account := game.guest(t)
	require.NoError(t, game.click(t, token))

	banned, err := antibotv1connect.NewAdminServiceClient(http.DefaultClient, game.consoleURL).BanPlayer(t.Context(),
		connect.NewRequest(&antibotv1.BanPlayerRequest{AccountId: account, Duration: durationpb.New(time.Hour)}))
	require.NoError(t, err)
	require.True(t, banned.Msg.GetEnforced())

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var refused *connect.Error
		if !assert.ErrorAs(c, game.click(t, token), &refused) {
			return
		}
		assert.Equal(c, connect.CodePermissionDenied, refused.Code())
		require.Len(c, refused.Details(), 1)
		detail, err := refused.Details()[0].Value()
		require.NoError(c, err)
		refusal, ok := detail.(*planetv1.BanRefusal)
		require.True(c, ok)
		assert.Equal(c, banned.Msg.GetBannedUntil().AsTime().UnixMilli(), refusal.GetBannedUntilUnixMs())
	}, 10*time.Second, 50*time.Millisecond, "the ban travels through the antibot's outbox to planet's refused callers")

	_, err = planetv1connect.NewAdminServiceClient(http.DefaultClient, game.consoleURL).UnbanPlayer(t.Context(),
		connect.NewRequest(&planetv1.UnbanPlayerRequest{AccountId: account}))
	require.NoError(t, err, "the old path is answered by the antibot until the backoffice moves")

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.NoError(c, game.click(t, token))
	}, 10*time.Second, 50*time.Millisecond, "the unban admits the caller again")
}

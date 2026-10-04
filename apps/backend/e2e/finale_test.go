package e2e_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet"
	"github.com/raphoester/clickplanet.lol-backend/internal/player"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type finale struct {
	gameStack

	startsAt time.Time
	endsAt   time.Time
}

func startFinale(t *testing.T, startsIn, lasts time.Duration) finale {
	t.Helper()

	postgres := cppg.StartTestServer(t)
	secret, _ := cpsession.TestKeyPair()
	public, internal, admin := listen(t), listen(t), listen(t)
	server := cpbootstrap.ServerConfig{
		BindAddress:         public.Addr().String(),
		InternalBindAddress: internal.Addr().String(),
		AdminBindAddress:    admin.Addr().String(),
	}

	authConfig := auth.Config{
		SignerConfig: cpsession.SignerConfig{Enabled: true, Secret: secret, TTL: time.Hour},
		Database:     postgres.ConfigFor("auth"),
	}
	authConfig.RateLimiter.PerSecond = 100
	authConfig.RateLimiter.Burst = 100

	planetConfig := planet.Config{Database: postgres.ConfigFor("planet")}
	planetConfig.GameMap.MaxIndex = mapTiles
	planetConfig.Auth = cpsession.VerifierConfig{Enabled: true, Enforce: true}
	planetConfig.RateLimiter.PerSecond = 0.2
	planetConfig.RateLimiter.Burst = 60
	planetConfig.RateLimiter.ScopeMultiplier = 10

	chatConfig := chat.Config{Database: postgres.ConfigFor("chat")}

	startsAt := time.Now().Add(startsIn).Truncate(time.Millisecond)
	endsAt := startsAt.Add(lasts)
	seasonsConfig := seasons.Config{Database: postgres.ConfigFor("seasons")}
	seasonsConfig.Calendar.List = slices.Grow(seasonsConfig.Calendar.List, 1)[:1]
	seasonsConfig.Calendar.List[0].EndsAt = endsAt
	seasonsConfig.Calendar.List[0].Finale = lasts
	seasonsConfig.Finale.CheckEvery = 20 * time.Millisecond
	seasonsConfig.Lead.Margin = 1
	seasonsConfig.Lead.Hold = time.Millisecond

	authModule, fakes := auth.NewModuleWithFakeProviders(authConfig)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- cpbootstrap.RunOn(ctx, cpbootstrap.Options{
			Server:         server,
			Logger:         slog.New(slog.DiscardHandler),
			StartupTimeout: time.Minute,
			Modules: []cpbootstrap.Module{
				authModule,
				planet.NewModule(planetConfig),
				player.NewModule(player.Config{Database: postgres.ConfigFor("player"), TagSalt: "pepper"}),
				chat.NewModule(chatConfig),
				seasons.NewModule(seasonsConfig),
			},
		}, public, internal, admin)
	}()
	t.Cleanup(func() {
		cancel()
		assert.NoError(t, <-done)
	})

	waitUntilServed(t, server.BindAddress)

	return finale{
		gameStack: gameStack{baseURL: "http://" + server.BindAddress, adminURL: "http://" + server.AdminBindAddress, fakes: fakes},
		startsAt:  startsAt,
		endsAt:    endsAt,
	}
}

func (p *gamer) clickFor(tile uint32, country string) (*planetv1.ClickResponse, error) {
	p.t.Helper()

	req := connect.NewRequest(&planetv1.ClickRequest{TileId: tile, CountryId: country})
	p.send(req.Header())
	res, err := planetv1connect.NewClickServiceClient(http.DefaultClient, p.stack.baseURL).Click(p.t.Context(), req)
	if err != nil {
		return nil, err //nolint:wrapcheck // the test reads the Connect code.
	}
	return res.Msg, nil
}

func (p *gamer) charges() *planetv1.ChargesHeld {
	p.t.Helper()

	req := connect.NewRequest(&planetv1.GetChargesRequest{})
	p.send(req.Header())
	res, err := planetv1connect.NewClickServiceClient(http.DefaultClient, p.stack.baseURL).GetCharges(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg.GetCharges()
}

func (p *gamer) budget(country string) *planetv1.ClickBudget {
	p.t.Helper()

	req := connect.NewRequest(&planetv1.GetBudgetRequest{CountryId: country})
	p.send(req.Header())
	res, err := planetv1connect.NewClickServiceClient(http.DefaultClient, p.stack.baseURL).GetBudget(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg.GetBudget()
}

func (s gameStack) announcements(t *testing.T, kind string) []map[string]any {
	t.Helper()

	res, err := chatv1connect.NewChatServiceClient(http.DefaultClient, s.baseURL).
		GetHistory(t.Context(), connect.NewRequest(&chatv1.GetHistoryRequest{}))
	require.NoError(t, err)

	var payloads []map[string]any
	for _, announcement := range res.Msg.GetAnnouncements() {
		if announcement.GetKind() != kind {
			continue
		}
		var payload map[string]any
		require.NoError(t, json.Unmarshal([]byte(announcement.GetPayload()), &payload))
		payloads = append(payloads, payload)
	}
	return payloads
}

func frozen(err error) bool {
	var connectErr *connect.Error
	if connect.CodeOf(err) != connect.CodeFailedPrecondition || !errors.As(err, &connectErr) {
		return false
	}
	for _, detail := range connectErr.Details() {
		if value, valueErr := detail.Value(); valueErr == nil {
			if _, ok := value.(*planetv1.MapFrozen); ok {
				return true
			}
		}
	}
	return false
}

func TestTheFinaleGivesABankAndABombOnceThenTheMapFreezesAtTheEnd(t *testing.T) {
	game := startFinale(t, 15*time.Second, 4*time.Second)
	ada := game.newPlayer(t)
	bob := game.newPlayer(t)
	require.True(t, time.Now().Before(game.startsAt), "the boot took longer than the test leaves it")

	res, err := ada.clickFor(1000, "fr")
	require.NoError(t, err)
	require.False(t, res.GetGift(), "no gift before the finale")
	require.False(t, ada.charges().GetBomb())

	time.Sleep(time.Until(game.startsAt))

	var gifted *planetv1.ClickResponse
	require.Eventually(t, func() bool {
		gifted, err = ada.clickFor(1001, "fr")
		return err == nil && gifted.GetGift()
	}, 5*time.Second, 25*time.Millisecond, "the first click of the finale brings the gift")

	assert.True(t, ada.charges().GetBomb(), "the gift is a bomb")
	budget := ada.budget("fr")
	assert.InDelta(t, float64(budget.GetCapacity()), budget.GetTokens(), 0.5, "and a full bank")
	assert.InDelta(t, 3*0.2, budget.GetRefillPerSecond(), 1e-6, "at three times the pace")

	res, err = ada.clickFor(1002, "fr")
	require.NoError(t, err)
	assert.False(t, res.GetGift(), "once per finale")

	time.Sleep(100 * time.Millisecond)
	for tile := uint32(2000); tile < 2004; tile++ {
		_, err := bob.clickFor(tile, "dz")
		require.NoError(t, err)
	}

	require.Eventually(t, func() bool {
		changes := game.announcements(t, "lead_changed")
		return len(changes) == 1 && changes[0]["leader"] == "dz" && changes[0]["passed"] == "fr"
	}, 5*time.Second, 25*time.Millisecond, "Algeria passes France")

	time.Sleep(time.Until(game.endsAt))

	require.Eventually(t, func() bool {
		_, err := ada.clickFor(1003, "fr")
		return frozen(err)
	}, 5*time.Second, 25*time.Millisecond, "a click after the end is refused as frozen, not throttled")

	drop := connect.NewRequest(&planetv1.DropBombRequest{Target: &planetv1.GlobePoint{X: 1}, CountryId: "fr"})
	ada.send(drop.Header())
	_, err = planetv1connect.NewClickServiceClient(http.DefaultClient, game.baseURL).DropBomb(t.Context(), drop)
	assert.True(t, frozen(err), "a bomb is refused too, and kept")
	assert.True(t, ada.charges().GetBomb())

	require.Eventually(t, func() bool {
		wins := game.announcements(t, "season_won")
		return len(wins) == 1 && wins[0]["winner"] == "dz" && wins[0]["season"] == float64(0)
	}, 5*time.Second, 25*time.Millisecond, "Algeria wins Season 0")
}

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

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet"
	"github.com/raphoester/clickplanet.lol-backend/internal/player"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// The tile count of the embedded map: planet refuses to boot on any other.
const mapTiles = 257948

type gameStack struct {
	baseURL string
}

// startGame boots auth, planet and player on one test postgres, with the internal listener the two last ones dial.
func startGame(t *testing.T) gameStack {
	t.Helper()

	postgres := cppg.StartTestServer(t)
	secret, _ := cpsession.TestKeyPair()
	server := cpbootstrap.ServerConfig{BindAddress: freeAddress(t), InternalBindAddress: freeAddress(t)}

	authConfig := auth.Config{
		SignerConfig: cpsession.SignerConfig{Enabled: true, Secret: secret, TTL: time.Hour},
		Database:     postgres.ConfigFor("auth"),
	}
	authConfig.RateLimiter.PerSecond = 100
	authConfig.RateLimiter.Burst = 100

	planetConfig := planet.Config{Database: postgres.ConfigFor("planet")}
	planetConfig.GameMap.MaxIndex = mapTiles
	planetConfig.Auth = cpsession.VerifierConfig{Enabled: true, Enforce: true}
	planetConfig.RateLimiter.PerSecond = 100
	planetConfig.RateLimiter.Burst = 100
	planetConfig.RateLimiter.ScopeMultiplier = 1

	playerConfig := player.Config{Enabled: true, Database: postgres.ConfigFor("player")}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- cpbootstrap.Run(ctx, cpbootstrap.Options{
			Server:         server,
			Logger:         slog.New(slog.DiscardHandler),
			StartupTimeout: time.Minute,
			Modules: []cpbootstrap.Module{
				auth.NewModule(authConfig), planet.NewModule(planetConfig), player.NewModule(playerConfig),
			},
		})
	}()
	t.Cleanup(func() {
		cancel()
		assert.NoError(t, <-done)
	})

	require.Eventually(t, func() bool {
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", server.BindAddress)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, time.Minute, 50*time.Millisecond, "the server never came up")

	return gameStack{baseURL: "http://" + server.BindAddress}
}

// gamer is one browser: the cookie auth set, and the click token minted with it.
type gamer struct {
	t      *testing.T
	stack  gameStack
	cookie string
	token  string
}

func (s gameStack) newPlayer(t *testing.T) *gamer {
	t.Helper()

	req := connect.NewRequest(&authv1.CreateSessionRequest{AttestationToken: "unused"})
	req.Header().Set("X-Real-IP", callerIP)
	res, err := authv1connect.NewAuthServiceClient(http.DefaultClient, s.baseURL).CreateSession(t.Context(), req)
	require.NoError(t, err)
	cookie, err := http.ParseSetCookie(res.Header().Get("Set-Cookie"))
	require.NoError(t, err)

	return &gamer{t: t, stack: s, cookie: cookie.Name + "=" + cookie.Value, token: res.Msg.GetToken()}
}

func (p *gamer) send(header http.Header) {
	header.Set("X-Real-IP", callerIP)
	header.Set(cpconnect.SessionHeader, p.token)
	header.Set("Cookie", p.cookie)
}

func (p *gamer) click(tile uint32, country string) {
	p.t.Helper()

	req := connect.NewRequest(&planetv1.ClickRequest{TileId: tile, CountryId: country})
	p.send(req.Header())
	_, err := planetv1connect.NewClickServiceClient(http.DefaultClient, p.stack.baseURL).Click(p.t.Context(), req)
	require.NoError(p.t, err)
}

func (p *gamer) players() playerv1connect.PlayerServiceClient {
	return playerv1connect.NewPlayerServiceClient(http.DefaultClient, p.stack.baseURL)
}

func (p *gamer) stats() *playerv1.Stats {
	p.t.Helper()

	req := connect.NewRequest(&playerv1.GetStatsRequest{})
	p.send(req.Header())
	res, err := p.players().GetStats(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg.GetStats()
}

func TestATileTakenWithAnAccountCountsOnItsStats(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)

	ada.click(1, "fr")
	ada.click(2, "fr")
	ada.click(2, "fr")

	require.Eventually(t, func() bool { return ada.stats().GetTilesTaken() == 2 }, 5*time.Second, 20*time.Millisecond,
		"a click on a tile already held takes nothing")
	stats := ada.stats()
	assert.Equal(t, uint32(1), stats.GetStreakCurrent())
	assert.Equal(t, time.Now().UTC().Format(time.DateOnly), stats.GetStreakLastDay())

	bob := game.newPlayer(t)
	assert.Zero(t, bob.stats().GetTilesTaken(), "another account's takes are not its own")
}

func TestANameIsSetAndReadBack(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)

	set := connect.NewRequest(&playerv1.SetNameRequest{Name: "  Ada\n"})
	ada.send(set.Header())
	_, err := ada.players().SetName(t.Context(), set)
	require.NoError(t, err)

	get := connect.NewRequest(&playerv1.GetProfileRequest{})
	ada.send(get.Header())
	res, err := ada.players().GetProfile(t.Context(), get)
	require.NoError(t, err)
	assert.Equal(t, "Ada", res.Msg.GetProfile().GetName())
}

func TestAPlayerCallWithNoTokenIsUnauthenticated(t *testing.T) {
	game := startGame(t)

	_, err := playerv1connect.NewPlayerServiceClient(http.DefaultClient, game.baseURL).
		GetStats(t.Context(), connect.NewRequest(&playerv1.GetStatsRequest{}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestADeletedAccountLosesItsStatsAndItsName(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ada.click(1, "fr")
	set := connect.NewRequest(&playerv1.SetNameRequest{Name: "Ada"})
	ada.send(set.Header())
	_, err := ada.players().SetName(t.Context(), set)
	require.NoError(t, err)
	require.Eventually(t, func() bool { return ada.stats().GetTilesTaken() == 1 }, 5*time.Second, 20*time.Millisecond)

	deletion := connect.NewRequest(&authv1.DeleteAccountRequest{})
	ada.send(deletion.Header())
	_, err = authv1connect.NewAuthServiceClient(http.DefaultClient, game.baseURL).DeleteAccount(t.Context(), deletion)
	require.NoError(t, err)

	// The token outlives the account until it expires, so it still reads what is left: nothing.
	get := connect.NewRequest(&playerv1.GetProfileRequest{})
	ada.send(get.Header())
	require.Eventually(t, func() bool {
		res, err := ada.players().GetProfile(t.Context(), get)
		return err == nil && res.Msg.GetProfile().GetName() == ""
	}, 5*time.Second, 20*time.Millisecond)
	assert.Zero(t, ada.stats().GetTilesTaken())
}

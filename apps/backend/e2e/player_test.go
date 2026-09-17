package e2e_test

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
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
	"github.com/raphoester/clickplanet.lol-backend/internal/chat"
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
	fakes   auth.FakeProviders
}

// startGame boots auth, planet, player and chat on one test postgres, with the internal listener the last three
// dial. Auth signs in with fake providers.
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

	playerConfig := player.Config{Database: postgres.ConfigFor("player"), TagSalt: "pepper"}

	chatConfig := chat.Config{Database: postgres.ConfigFor("chat")}
	chatConfig.RateLimiter.PerSecond = 100
	chatConfig.RateLimiter.Burst = 100

	authModule, fakes := auth.NewModuleWithFakeProviders(authConfig)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- cpbootstrap.Run(ctx, cpbootstrap.Options{
			Server:         server,
			Logger:         slog.New(slog.DiscardHandler),
			StartupTimeout: time.Minute,
			Modules: []cpbootstrap.Module{
				authModule, planet.NewModule(planetConfig), player.NewModule(playerConfig), chat.NewModule(chatConfig),
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

	return gameStack{baseURL: "http://" + server.BindAddress, fakes: fakes}
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

// link signs the guest in with a provider, which links the identity to its account, and mints again so the
// token is the linked account's.
func (p *gamer) link(subject string) {
	p.t.Helper()

	client := authv1connect.NewAuthServiceClient(http.DefaultClient, p.stack.baseURL)

	start := connect.NewRequest(&authv1.StartSignInRequest{
		Provider: authv1.Provider_PROVIDER_GOOGLE, Intent: authv1.SignInIntent_SIGN_IN_INTENT_LINK,
	})
	p.send(start.Header())
	started, err := client.StartSignIn(p.t.Context(), start)
	require.NoError(p.t, err)
	flow, err := http.ParseSetCookie(started.Header().Get("Set-Cookie"))
	require.NoError(p.t, err)
	authorization, err := url.Parse(started.Msg.GetAuthorizationUrl())
	require.NoError(p.t, err)

	p.stack.fakes.Google.Grant(subject, auth.Claim{Subject: subject})
	complete := connect.NewRequest(&authv1.CompleteSignInRequest{Code: subject, State: authorization.Query().Get("state")})
	p.send(complete.Header())
	complete.Header().Set("Cookie", p.cookie+"; "+flow.Name+"="+flow.Value)
	completed, err := client.CompleteSignIn(p.t.Context(), complete)
	require.NoError(p.t, err)
	require.Equal(p.t, authv1.SignInOutcome_SIGN_IN_OUTCOME_LINKED, completed.Msg.GetOutcome())

	for _, line := range completed.Header().Values("Set-Cookie") {
		cookie, err := http.ParseSetCookie(line)
		require.NoError(p.t, err)
		if cookie.MaxAge >= 0 {
			p.cookie = cookie.Name + "=" + cookie.Value
		}
	}

	mint := connect.NewRequest(&authv1.CreateSessionRequest{AttestationToken: "unused"})
	p.send(mint.Header())
	minted, err := client.CreateSession(p.t.Context(), mint)
	require.NoError(p.t, err)
	p.token = minted.Msg.GetToken()
}

func (p *gamer) setName(name string) (*playerv1.Profile, error) {
	p.t.Helper()

	req := connect.NewRequest(&playerv1.SetNameRequest{Name: name})
	p.send(req.Header())
	res, err := p.players().SetName(p.t.Context(), req)
	if err != nil {
		return nil, fmt.Errorf("SetName failed: %w", err)
	}
	return res.Msg.GetProfile(), nil
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

func TestALinkedPlayerSetsAUsernameAndReadsItBack(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ada.link("google-ada")

	_, err := ada.setName("Ada_L")
	require.NoError(t, err)

	get := connect.NewRequest(&playerv1.GetProfileRequest{})
	ada.send(get.Header())
	res, err := ada.players().GetProfile(t.Context(), get)
	require.NoError(t, err)
	assert.Equal(t, "Ada_L", res.Msg.GetProfile().GetName())
}

func TestAGuestMayNotChooseAUsername(t *testing.T) {
	game := startGame(t)

	_, err := game.newPlayer(t).setName("Ada_L")

	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}

func TestAUsernameAnotherPlayerHoldsIsAlreadyExists(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ada.link("google-ada")
	_, err := ada.setName("Ada_L")
	require.NoError(t, err)

	bob := game.newPlayer(t)
	bob.link("google-bob")
	_, err = bob.setName("ada_l")

	assert.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(err))
	_, err = ada.setName("ADA_L")
	assert.NoError(t, err, "a player sets its own name again in another case")
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
	ada.link("google-ada")
	ada.click(1, "fr")
	_, err := ada.setName("Ada")
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

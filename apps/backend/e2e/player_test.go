package e2e_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
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
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

const mapTiles = 262119

type gameStack struct {
	baseURL  string
	adminURL string
	fakes    auth.FakeProviders
	postgres *cppg.TestServer
}

func startGame(t *testing.T) gameStack {
	t.Helper()

	game, _ := startGameOn(t, cppg.StartTestServer(t))
	return game
}

func startGameOn(t *testing.T, postgres *cppg.TestServer) (gameStack, func()) {
	t.Helper()

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
	planetConfig.RateLimiter.PerSecond = 100
	planetConfig.RateLimiter.Burst = 100
	planetConfig.RateLimiter.ScopeMultiplier = 1
	planetConfig.LedgerStorage.FlushInterval = 50 * time.Millisecond

	playerConfig := player.Config{Database: postgres.ConfigFor("player"), TagSalt: "pepper"}
	playerConfig.Takes.PollInterval = 20 * time.Millisecond

	seasonsConfig := currentSeason(t, postgres.ConfigFor("seasons"))
	seasonsConfig.Takes.PollInterval = 20 * time.Millisecond

	chatConfig := chat.Config{Database: postgres.ConfigFor("chat")}
	chatConfig.RateLimiter.PerSecond = 100
	chatConfig.RateLimiter.Burst = 100

	authModule, fakes := auth.NewModuleWithFakeProviders(authConfig)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- cpbootstrap.RunOn(ctx, cpbootstrap.Options{
			Server:         server,
			Logger:         slog.New(slog.DiscardHandler),
			StartupTimeout: time.Minute,
			Modules: []cpbootstrap.Module{
				authModule, planet.NewModule(planetConfig), player.NewModule(playerConfig), chat.NewModule(chatConfig),
				seasons.NewModule(seasonsConfig),
			},
		}, public, internal, admin)
	}()
	var once sync.Once
	stop := func() {
		once.Do(func() {
			cancel()
			assert.NoError(t, <-done)
		})
	}
	t.Cleanup(stop)

	waitUntilServed(t, server.BindAddress)

	return gameStack{
		baseURL: "http://" + server.BindAddress, adminURL: "http://" + server.AdminBindAddress, fakes: fakes, postgres: postgres,
	}, stop
}

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

func (p *gamer) link(subject string) {
	p.t.Helper()

	p.signIn(subject, authv1.SignInIntent_SIGN_IN_INTENT_LINK, authv1.SignInOutcome_SIGN_IN_OUTCOME_LINKED)
}

func (p *gamer) signIn(subject string, intent authv1.SignInIntent, outcome authv1.SignInOutcome) {
	p.t.Helper()

	client := authv1connect.NewAuthServiceClient(http.DefaultClient, p.stack.baseURL)

	start := connect.NewRequest(&authv1.StartSignInRequest{
		Provider: authv1.Provider_PROVIDER_GOOGLE, Intent: intent,
	})
	p.send(start.Header())
	started, err := client.StartSignIn(p.t.Context(), start)
	require.NoError(p.t, err)
	flow, err := http.ParseSetCookie(started.Header().Get("Set-Cookie"))
	require.NoError(p.t, err)
	authorization, err := url.Parse(started.Msg.GetAuthorizationUrl())
	require.NoError(p.t, err)

	p.stack.fakes.Google.Grant(subject, auth.ClaimOf(subject, "", false))
	complete := connect.NewRequest(&authv1.CompleteSignInRequest{Code: subject, State: authorization.Query().Get("state")})
	p.send(complete.Header())
	complete.Header().Set("Cookie", p.cookie+"; "+flow.Name+"="+flow.Value)
	completed, err := client.CompleteSignIn(p.t.Context(), complete)
	require.NoError(p.t, err)
	require.Equal(p.t, outcome, completed.Msg.GetOutcome())

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

func (p *gamer) setColor(color playerv1.NameColor) error {
	p.t.Helper()

	req := connect.NewRequest(&playerv1.SetColorRequest{Color: color})
	p.send(req.Header())
	if _, err := p.players().SetColor(p.t.Context(), req); err != nil {
		return fmt.Errorf("SetColor failed: %w", err)
	}
	return nil
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

func TestAnybodyReadsAPlayerByItsUsernameWithNoToken(t *testing.T) {
	game := startGame(t)
	before := time.Now().Add(-time.Second)
	ada := game.newPlayer(t)
	ada.link("google-ada")
	_, err := ada.setName("Ada_L")
	require.NoError(t, err)
	ada.click(1, "fr")
	require.Eventually(t, func() bool { return ada.stats().GetTilesTaken() == 1 }, 5*time.Second, 20*time.Millisecond)

	anybody := playerv1connect.NewPlayerServiceClient(http.DefaultClient, game.baseURL, connect.WithHTTPGet())
	res, err := anybody.GetPlayer(t.Context(), connect.NewRequest(&playerv1.GetPlayerRequest{Name: "ada_l"}))

	require.NoError(t, err)
	player := res.Msg.GetPlayer()
	assert.Equal(t, "Ada_L", player.GetName())
	assert.Equal(t, uint64(1), player.GetStats().GetTilesTaken())
	assert.Equal(t, uint32(1), player.GetStats().GetStreakCurrent())
	assert.WithinRange(t, time.UnixMilli(player.GetCreatedAtUnixMs()), before, time.Now(), "auth says when the guest was made")

	_, err = anybody.GetPlayer(t.Context(), connect.NewRequest(&playerv1.GetPlayerRequest{Name: "Bob"}))
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
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

	get := connect.NewRequest(&playerv1.GetProfileRequest{})
	ada.send(get.Header())
	require.Eventually(t, func() bool {
		res, err := ada.players().GetProfile(t.Context(), get)
		return err == nil && res.Msg.GetProfile().GetName() == ""
	}, 5*time.Second, 20*time.Millisecond)
	assert.Zero(t, ada.stats().GetTilesTaken())
}

func TestAnOperatorReconcilesTheTitlesOnTheAdminListenerOnly(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ada.link("google-ada")
	_, err := ada.setName("Ada")
	require.NoError(t, err)
	ada.click(1, "fr")
	guest := game.newPlayer(t)
	guest.click(2, "fr")
	require.Eventually(t, func() bool {
		return ada.stats().GetTilesTaken() == 1 && guest.stats().GetTilesTaken() == 1
	}, 5*time.Second, 20*time.Millisecond)

	request := connect.NewRequest(&playerv1.ReconcileTitlesRequest{})
	_, err = playerv1connect.NewAdminServiceClient(http.DefaultClient, game.baseURL).ReconcileTitles(t.Context(), request)
	require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err), "the public router does not serve it")

	admin := playerv1connect.NewAdminServiceClient(http.DefaultClient, game.adminURL)
	first, err := admin.ReconcileTitles(t.Context(), request)
	require.NoError(t, err)
	assert.Zero(t, first.Msg.GetRevoked(), "the worker granted the guest nothing, and the player only what it earns")

	second, err := admin.ReconcileTitles(t.Context(), request)
	require.NoError(t, err)
	assert.Zero(t, second.Msg.GetGranted())
	assert.Zero(t, second.Msg.GetRevoked())
}

func TestASignedInStreamHearsTheTitleItsTakesEarnAndTheTitleCanBeWorn(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ada.link("google-ada")

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	listen := connect.NewRequest(&playerv1.ListenForEventsRequest{})
	ada.send(listen.Header())
	stream, err := ada.players().ListenForEvents(ctx, listen)
	require.NoError(t, err)
	require.True(t, stream.Receive(), "the roster comes first")

	earned := make(chan string, 16)
	go func() {
		for stream.Receive() {
			if title := stream.Msg().GetTitleEarned(); title != nil {
				earned <- title.GetTitle().GetId()
			}
		}
	}()

	for tile := range uint32(100) {
		ada.click(tile+1, "fr")
	}

	require.Eventually(t, func() bool {
		select {
		case id := <-earned:
			return id == "settler"
		default:
			return false
		}
	}, 10*time.Second, 20*time.Millisecond, "the hundredth take earns Settler, live")

	get := connect.NewRequest(&playerv1.GetTitlesRequest{})
	ada.send(get.Header())
	dashboard, err := ada.players().GetTitles(t.Context(), get)
	require.NoError(t, err)
	conquest := dashboard.Msg.GetTracks()[0]
	assert.Equal(t, "conquest", conquest.GetId())
	assert.Equal(t, uint64(100), conquest.GetProgress())
	assert.True(t, conquest.GetSteps()[0].GetEarned())

	wear := connect.NewRequest(&playerv1.WearTitleRequest{TitleId: "settler"})
	ada.send(wear.Header())
	worn, err := ada.players().WearTitle(t.Context(), wear)
	require.NoError(t, err)
	assert.Equal(t, "settler", worn.Msg.GetWorn().GetId())
	assert.Equal(t, uint32(1), worn.Msg.GetWorn().GetRank().GetNumber())

	refused := connect.NewRequest(&playerv1.WearTitleRequest{TitleId: "warmaster"})
	ada.send(refused.Header())
	_, err = ada.players().WearTitle(t.Context(), refused)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestAWornTitleShowsOnTheRosterAtOnceAndOnTheChat(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ada.link("google-ada")
	_, err := ada.setName("Ada_L")
	require.NoError(t, err)
	require.NoError(t, ada.announce("fr"))

	for tile := range uint32(100) {
		ada.click(tile+1, "fr")
	}
	require.Eventually(t, func() bool {
		get := connect.NewRequest(&playerv1.GetTitlesRequest{})
		ada.send(get.Header())
		dashboard, err := ada.players().GetTitles(t.Context(), get)
		return err == nil && dashboard.Msg.GetWorn().GetId() == "settler"
	}, 10*time.Second, 20*time.Millisecond, "the hundredth take earns Settler")
	require.Nil(t, game.roster(t).Msg.GetEntries()[0].GetWornTitle(), "announced before it held a title")

	wear := connect.NewRequest(&playerv1.WearTitleRequest{TitleId: "settler"})
	ada.send(wear.Header())
	_, err = ada.players().WearTitle(t.Context(), wear)
	require.NoError(t, err)

	entries := game.roster(t).Msg.GetEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "settler", entries[0].GetWornTitle().GetId())

	message, err := ada.post()
	require.NoError(t, err)
	assert.Equal(t, "settler", message.GetAuthorTitle().GetId())
	assert.Equal(t, "conquest", message.GetAuthorTitle().GetRank().GetTrackId())
	assert.Equal(t, "settler", ada.history()[0].GetAuthorTitle().GetId())
}

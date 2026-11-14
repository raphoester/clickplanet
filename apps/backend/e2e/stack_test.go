package e2e_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1/gamemasterv1connect"
	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1/scoringv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster"
	"github.com/raphoester/clickplanet.lol-backend/internal/mcp"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet"
	"github.com/raphoester/clickplanet.lol-backend/internal/player"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpanthropic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis/cpredistest"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/support"
)

const (
	deletionGrace = 3 * time.Second
	callerIP      = "203.0.113.7"
	adminOrigin   = "https://backoffice.clickplanet.lol"
	mediaOrigin   = "https://media.example"
	mapTiles      = 260503
	// The fake Messages API's: it checks none.
	judgeKey = "sk-ant-e2e-key"
)

// dailyCutoff is production's 21:00 unless a test moves it before it starts the game.
var dailyCutoff = 21 * time.Hour

type gameStack struct {
	baseURL    string
	consoleURL string
	adminURL   string
	fakes      auth.FakeProviders
	postgres   *cppgtest.Server
	missions   *cppg.Postgres
	redis      cpredis.Config
}

func startGame(t *testing.T) gameStack {
	t.Helper()

	postgres := cppgtest.StartServer(t)
	redis := cpredistest.StartServer(t).Config()
	secret, _ := cpsession.TestKeyPair()
	public, internal, console, admin := listen(t), listen(t), listen(t), listen(t)
	server := cpbootstrap.ServerConfig{
		BindAddress:         public.Addr().String(),
		InternalBindAddress: internal.Addr().String(),
		ConsoleBindAddress:  console.Addr().String(),
		AdminBindAddress:    admin.Addr().String(),
	}

	authConfig := auth.Config{
		SignerConfig: cpsession.SignerConfig{Enabled: true, Secret: secret, TTL: time.Hour},
		Database:     postgres.ConfigFor("auth"),
		Redis:        redis,
	}
	authConfig.RateLimiter.PerSecond = 100
	authConfig.RateLimiter.Burst = 100
	// Short enough for a test to see an account purged, long enough to see it kept first.
	authConfig.Deletion.Grace = deletionGrace
	authConfig.Deletion.Interval = 50 * time.Millisecond
	authConfig.Admin.Origin = adminOrigin
	authConfig.Connector = auth.ConnectorConfig{Enabled: true, Issuer: "http://" + server.AdminBindAddress}

	planetConfig := planet.Config{Database: postgres.ConfigFor("planet"), Redis: redis}
	clickAtWill(t, planetConfig.Database)
	planetConfig.GameMap.MaxIndex = mapTiles
	planetConfig.Auth = cpsession.VerifierConfig{Enabled: true}

	antibotConfig := antibot.Config{Enabled: true, Database: postgres.ConfigFor("antibot"), Redis: redis}
	watchNoOne(t, antibotConfig.Database)

	playerConfig := player.Config{Database: postgres.ConfigFor("player"), Redis: redis, TagSalt: "pepper"}

	progressionConfig := progression.Config{Database: postgres.ConfigFor("progression"), Redis: redis}

	scoringConfig := scoring.Config{Database: postgres.ConfigFor("scoring"), Redis: redis}

	missionsConfig := missions.Config{Database: postgres.ConfigFor("missions"), Redis: redis}

	bonusesConfig := bonuses.Config{Database: postgres.ConfigFor("bonuses"), Redis: redis}

	quizzesConfig := quizzes.Config{Database: postgres.ConfigFor("quizzes"), Redis: redis}

	gamemasterConfig := gamemaster.Config{Database: postgres.ConfigFor("gamemaster"), Redis: redis, DailyCutoff: dailyCutoff}
	seasonRuns(t, gamemasterConfig.Database)

	chatConfig := chat.Config{Database: postgres.ConfigFor("chat"), Redis: redis}
	chatConfig.RateLimiter.PerSecond = 100
	chatConfig.RateLimiter.Burst = 100
	chatConfig.Judge.Anthropic = cpanthropic.Config{APIKey: judgeKey, BaseURL: startJudge(t)}

	bucket := startBucket(t)
	supportConfig := support.Config{Database: postgres.ConfigFor("support"), Redis: redis}
	supportConfig.Attachments.Enabled = true
	supportConfig.Attachments.Bucket.Endpoint = bucket.url
	supportConfig.Attachments.Bucket.Region = "auto"
	supportConfig.Attachments.Bucket.Name = "support"
	supportConfig.Attachments.Bucket.AccessKeyID = "the-key-id"
	supportConfig.Attachments.Bucket.SecretAccessKey = "the-secret-key"

	promoConfig := promo.Config{Database: postgres.ConfigFor("promo")}
	promoConfig.Backgrounds.Enabled = true
	promoConfig.Backgrounds.PublicBaseURL = mediaOrigin
	promoConfig.Backgrounds.Bucket = supportConfig.Attachments.Bucket

	authModule, fakes := auth.NewModuleWithFakeProviders(authConfig)

	logger := slog.New(slog.DiscardHandler)
	gate, err := auth.NewAdminGate(auth.AdminConfig{Origin: adminOrigin}, server.InternalBindAddress, logger)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- cpbootstrap.RunOn(ctx, cpbootstrap.Options{
			Server: server,
			Logger: logger,
			Modules: []cpbootstrap.Module{
				authModule, planet.NewModule(planetConfig), bonuses.NewModule(bonusesConfig), quizzes.NewModule(quizzesConfig),
				antibot.NewModule(antibotConfig),
				player.NewModule(playerConfig),
				progression.NewModule(progressionConfig),
				chat.NewModule(chatConfig), scoring.NewModule(scoringConfig), support.NewModule(supportConfig),
				missions.NewModule(missionsConfig), gamemaster.NewModule(gamemasterConfig), promo.NewModule(promoConfig),
				mcp.NewModule(mcp.Config{Enabled: true, URL: "http://" + server.AdminBindAddress + "/mcp"}),
			},
			AdminGate: gate,
		}, public, internal, console, admin)
	}()
	t.Cleanup(func() {
		cancel()
		assert.NoError(t, <-done)
	})

	waitUntilServed(t, server.BindAddress)

	stack := gameStack{
		baseURL:    "http://" + server.BindAddress,
		consoleURL: "http://" + server.ConsoleBindAddress,
		adminURL:   "http://" + server.AdminBindAddress,
		fakes:      fakes,
		postgres:   postgres,
		redis:      redis,
	}
	stack.missions = stack.schema(t, "missions")
	stack.today(t)
	return stack
}

// Every test clicks from one address, so the rule set the stack plays lets it: written before planet boots.
func clickAtWill(t *testing.T, database cppg.Config) {
	t.Helper()

	require.NoError(t, planet.Migrate(t.Context(), database))
	db := cppg.New(database)
	require.NoError(t, db.ConnectCtx(t.Context()))
	defer func() { _ = db.Close() }()

	_, err := db.On(t.Context()).ExecContext(t.Context(), `
		UPDATE rule_sets
		SET rules = jsonb_set(rules, '{clicks}', rules->'clicks' ||
			'{"perSecond": 100, "burst": 1000, "scopeMultiplier": 1, "newAccountClicks": 1000}')
		WHERE name = 'default'
	`)
	require.NoError(t, err)
}

// Every test clicks from one address, faster than a hand: no watchdog may ban them. Bans by hand still apply.
func watchNoOne(t *testing.T, database cppg.Config) {
	t.Helper()

	require.NoError(t, antibot.Migrate(t.Context(), database))
	db := cppg.New(database)
	require.NoError(t, db.ConnectCtx(t.Context()))
	defer func() { _ = db.Close() }()

	_, err := db.On(t.Context()).ExecContext(t.Context(), `
		UPDATE rule_sets
		SET rules = (
			SELECT jsonb_object_agg(key, CASE WHEN value ? 'enabled' THEN value || '{"enabled": false}' ELSE value END)
			FROM jsonb_each(rules)
		)
		WHERE name = 'default'
	`)
	require.NoError(t, err)
}

func listen(t *testing.T) net.Listener {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	return listener
}

func waitUntilServed(t *testing.T, address string) {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+address+"/metrics", nil)
	require.NoError(t, err)
	// Pooled, it can come back while the next call dials, leaving a silent spare that holds Shutdown for 5s.
	req.Close = true
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "the server never came up")
	require.NoError(t, res.Body.Close())
}

// Season 0 ends 30 days on, 2 h past the cutoff, so its 2 h finale starts at the cutoff. The gamemaster tells scoring
// at boot.
func seasonRuns(t *testing.T, database cppg.Config) {
	t.Helper()

	require.NoError(t, gamemaster.Migrate(t.Context(), database))
	db := cppg.New(database)
	require.NoError(t, db.ConnectCtx(t.Context()))
	defer func() { _ = db.Close() }()
	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`UPDATE seasons SET length = $1::timestamptz - timestamptz '2026-10-08T00:00:00Z' WHERE number = 0`,
		time.Now().Add(30*24*time.Hour).UTC().Truncate(24*time.Hour).Add(dailyCutoff+2*time.Hour))
	require.NoError(t, err)
}

type today struct {
	book  string
	round string
}

// Today is the season's book and today's round, once the gamemaster opened both in scoring: a take counts in a book
// only once it is open.
func (s gameStack) today(t *testing.T) today {
	t.Helper()

	var found today
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		turn, err := gamemasterv1connect.NewGamemasterServiceClient(http.DefaultClient, s.baseURL).
			GetTurn(t.Context(), connect.NewRequest(&gamemasterv1.GetTurnRequest{}))
		require.NoError(c, err)
		found = today{book: turn.Msg.GetTimeline().GetCurrent().GetBookId(), round: turn.Msg.GetTimeline().GetToday().GetRoundId()}
		require.NotEmpty(c, found.book)
		require.NotEmpty(c, found.round)
		_, err = scoringv1connect.NewScoringServiceClient(http.DefaultClient, s.baseURL).
			GetRound(t.Context(), connect.NewRequest(&scoringv1.GetRoundRequest{RoundId: found.round}))
		require.NoError(c, err)
		var told bool
		require.NoError(c, s.missions.On(t.Context()).QueryRowContext(t.Context(),
			`SELECT EXISTS (SELECT 1 FROM mission_days WHERE day_id = $1)`, found.round).Scan(&told))
		require.True(c, told)
	}, 10*time.Second, 50*time.Millisecond, "the gamemaster opens the season's book and today's round in scoring, and tells missions the day")
	return found
}

func (s gameStack) schema(t *testing.T, name string) *cppg.Postgres {
	t.Helper()

	db := cppg.New(s.postgres.ConfigFor(name))
	require.NoError(t, db.ConnectCtx(t.Context()))
	t.Cleanup(func() { _ = db.Close() })
	return db
}

type storedFile struct {
	contentType string
	body        []byte
}

type fakeBucket struct {
	url string

	mu    sync.Mutex
	files map[string]storedFile
}

func startBucket(t *testing.T) *fakeBucket {
	t.Helper()

	bucket := &fakeBucket{files: map[string]storedFile{}}
	server := httptest.NewServer(http.HandlerFunc(bucket.serve))
	t.Cleanup(server.Close)
	bucket.url = server.URL
	return bucket
}

func (b *fakeBucket) serve(w http.ResponseWriter, r *http.Request) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if r.URL.Query().Get("X-Amz-Signature") == "" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodPut:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		b.files[r.URL.Path] = storedFile{contentType: r.Header.Get("Content-Type"), body: body}
	case http.MethodGet, http.MethodHead:
		file, kept := b.files[r.URL.Path]
		if !kept {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", file.contentType)
		w.Header().Set("Content-Length", strconv.Itoa(len(file.body)))
		if r.Method == http.MethodGet {
			_, _ = w.Write(file.body)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

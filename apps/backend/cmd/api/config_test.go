package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconfigs"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

func TestTheExampleConfigStillReachesTheStructs(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	require.True(t, config.AntiBot.Enabled)

	assert.Equal(t, 25, config.Planet.Bonus.Enclose.MaxTiles)
	assert.Equal(t, 8, config.Planet.Bonus.Spread.Clicks)
	assert.Equal(t, 4, config.Planet.Bonus.Spread.MaxPerBox)
	assert.Equal(t, 3, config.Planet.Bonus.Enclose.Held)
	assert.Equal(t, 3, config.Planet.Bonus.Enclose.MaxPerBox)
	assert.Equal(t, 30, config.Planet.Bonus.Shield.Held)
	assert.Equal(t, 5, config.Planet.Bonus.Shield.MinPerBox)
	assert.Equal(t, 20, config.Planet.Bonus.Shield.MaxPerBox)

	quiz := config.Planet.Bonus.Quiz
	assert.True(t, quiz.Enabled)
	assert.Equal(t, 4*time.Minute, quiz.MinInterval)
	assert.Equal(t, 8*time.Minute, quiz.MaxInterval)
	assert.Equal(t, 2*time.Minute, quiz.MissRetry)
	assert.Equal(t, 25*time.Second, quiz.OfferTTL)
	assert.Equal(t, 8*time.Second, quiz.AnswerWindow, "example.yaml must load the answerWindow it documents")
	assert.InDelta(t, 1.0, quiz.LeaderBias, 1e-9)
	assert.Equal(t, 12, quiz.MaxChargesPerHour)

	assert.Equal(t, 72*time.Hour, config.Planet.Ledger.Retention)
	assert.Equal(t, 5*time.Minute, config.Planet.Ledger.SweepInterval)
	assert.Equal(t, time.Second, config.Planet.LedgerStorage.FlushInterval)

	assert.False(t, config.AntiBot.Bans.Enforce, "the example must ship observing only")
	assert.Equal(t, []time.Duration{24 * time.Hour, 7 * 24 * time.Hour, 3 * 365 * 24 * time.Hour}, config.AntiBot.Bans.BanDurations)
	assert.Equal(t, "antibot", config.AntiBot.Database.Schema)
	assert.Equal(t, 5*time.Minute, config.AntiBot.Bans.ReflagInterval)

	assert.Equal(t, 2, config.AntiBot.Jury.MinSuspects)
	assert.Equal(t, 10*time.Minute, config.AntiBot.Jury.SuspicionWindow)

	assert.Equal(t, time.Minute, config.AntiBot.Evidence.SaveInterval)
	assert.Equal(t, 72*time.Hour, config.AntiBot.Evidence.Retention)

	require.True(t, config.AntiBot.Retaker.Enabled)
	assert.Equal(t, 5*time.Second, config.AntiBot.Retaker.Detector.ReactionWindow)
	assert.Equal(t, 12, config.AntiBot.Retaker.Detector.MinReactions)
	assert.Equal(t, 120*time.Millisecond, config.AntiBot.Retaker.Detector.MaxSpread)
	assert.Equal(t, 250*time.Millisecond, config.AntiBot.Retaker.Detector.MaxMedian)

	assert.Equal(t, map[string]int{"rush": 50}, config.AntiBot.Cohort.Detector.CrowdBySet,
		"players who start the Final Battle together are no crowd")

	require.True(t, config.AntiBot.Sequencer.Enabled)
	assert.Equal(t, 40, config.AntiBot.Sequencer.Detector.MinSteps)
	assert.InDelta(t, 0.75, config.AntiBot.Sequencer.Detector.MinShare, 1e-9)
	assert.Equal(t, 200, config.AntiBot.Sequencer.Detector.CertainSteps)
	assert.InDelta(t, 0.95, config.AntiBot.Sequencer.Detector.CertainShare, 1e-9)

	require.True(t, config.AntiBot.Metronome.Enabled)
	assert.Equal(t, 3*time.Second, config.AntiBot.Metronome.Detector.MaxGap)
	assert.Equal(t, 120*time.Millisecond, config.AntiBot.Metronome.Detector.MaxSpread)
	assert.Equal(t, 120, config.AntiBot.Metronome.Detector.MinClicks)
	assert.Equal(t, 30*time.Minute, config.AntiBot.Metronome.Detector.CertainFor)
	assert.Equal(t, 900, config.AntiBot.Metronome.Detector.CertainClicks)

	require.True(t, config.AntiBot.Scraper.Enabled)
	assert.InDelta(t, 5, config.AntiBot.Scraper.Detector.MinMaps, 1e-9)
	assert.InDelta(t, 15, config.AntiBot.Scraper.Detector.CertainMaps, 1e-9)
	assert.Equal(t, 15*time.Minute, config.AntiBot.Scraper.Detector.TrackWindow)

	churner := config.AntiBot.Churner
	require.True(t, churner.Enabled)
	assert.Equal(t, time.Hour, churner.Detector.Window)
	assert.Equal(t, 6, churner.Detector.V6.CertainAccounts)
	assert.Equal(t, 20, churner.Detector.V4.CertainAccounts)
	assert.Equal(t, 90*time.Second, churner.Detector.Relay.Handoff)
	assert.Equal(t, 32, churner.Detector.Relay.V6Bits)
	assert.Equal(t, 6, churner.Detector.Relay.CertainLinks)

	hopper := config.AntiBot.Hopper
	require.True(t, hopper.Enabled)
	assert.InDelta(t, 45, hopper.Detector.MinAngle, 1e-9)
	assert.Equal(t, 100*time.Millisecond, hopper.Detector.MinGap)
	assert.Equal(t, 30*time.Second, hopper.Detector.MaxGap)
	assert.Equal(t, 40, hopper.Detector.MinSteps)
	assert.InDelta(t, 0.4, hopper.Detector.MinShare, 1e-9)
	assert.Zero(t, hopper.Detector.CertainShare, "it only corroborates")

	rotator := config.AntiBot.Rotator
	require.True(t, rotator.Enabled)
	assert.Equal(t, 5*time.Minute, rotator.Detector.Slot)
	assert.Equal(t, 24*time.Hour, rotator.Detector.Window)
	assert.Equal(t, 15, rotator.Detector.MinSeconds)
	assert.InDelta(t, 90, rotator.Detector.MinByChance, 1e-9)
	assert.InDelta(t, 0.05, rotator.Detector.MaxShare, 1e-9)
	assert.Equal(t, 24, rotator.Detector.MinSlots)
	assert.Zero(t, rotator.Detector.CertainSlots, "it only corroborates")
}

func TestTheExampleConfigStillCarriesTheRestOfTheFile(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.Equal(t, "0.0.0.0:8080", config.HTTPServer.BindAddress)
	assert.Equal(t, time.Minute, config.HTTPServer.StartupTimeout)
	assert.NotZero(t, config.Planet.GameMap.MaxIndex)
	assert.InDelta(t, 0.2, config.Planet.RateLimiter.PerSecond, 1e-9, "one click every 5s")
	assert.Equal(t, 60, config.Planet.RateLimiter.Burst)
	require.Len(t, config.Planet.Toll.Steps, 3)
	assert.InDelta(t, 0.30, config.Planet.Toll.Steps[2].Share, 1e-9)
	assert.InDelta(t, 4, config.Planet.Toll.Steps[2].Slowdown, 1e-9)
	require.NoError(t, config.Planet.Validate())
	assert.Equal(t, time.Second, config.Planet.TilesStorage.FlushInterval)
	assert.Equal(t, "127.0.0.1:8081", config.HTTPServer.ConsoleBindAddress)
	assert.Equal(t, "127.0.0.1:8082", config.HTTPServer.InternalBindAddress)
	assert.Equal(t, "http://localhost:5173", config.HTTPServer.AllowedOrigin)
	assert.Equal(t, time.Hour, config.Auth.TTL)
}

func TestTheLocalStackConfigHasTheMapOfTheExampleAndAnInternalListener(t *testing.T) {
	var example, local Config
	require.NoError(t, cpconfigs.Load(&example, cpconfigs.FromFile("example.yaml")))
	require.NoError(t, cpconfigs.Load(&local, cpconfigs.FromFile("../../../../deploy/backend.yaml")))

	assert.Equal(t, example.Planet.GameMap.MaxIndex, local.Planet.GameMap.MaxIndex)
	assert.NotEmpty(t, local.HTTPServer.InternalBindAddress, "chat, player and seasons dial it while they are built")
}

func TestTheExampleConfigReachesTheBombSettings(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.InDelta(t, 4.0, config.Planet.Bonus.Bomb.Rings, 1e-9)
	assert.InDelta(t, 1.0, config.Planet.Bonus.Kinds["bomb"], 1e-9)
	require.NoError(t, config.Planet.Bonus.Validate())
}

func authBlock(t *testing.T, secret string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, fmt.Appendf(nil, `
httpServer:
  bindAddress: 0.0.0.0:8080
  internalBindAddress: 127.0.0.1:8082
  allowedOrigin: https://clickplanet.lol
gameMap:
  maxIndex: 100
database: {host: localhost, port: "5432", user: postgres, dbName: postgres, sslMode: disable, schema: planet}
redis: {address: localhost:6379}
chat:
  database: {host: localhost, port: "5432", user: postgres, dbName: postgres, sslMode: disable, schema: chat}
  redis: {address: localhost:6379}
player:
  database: {host: localhost, port: "5432", user: postgres, dbName: postgres, sslMode: disable, schema: player}
  redis: {address: localhost:6379}
seasons:
  database: {host: localhost, port: "5432", user: postgres, dbName: postgres, sslMode: disable, schema: seasons}
  redis: {address: localhost:6379}
missions:
  database: {host: localhost, port: "5432", user: postgres, dbName: postgres, sslMode: disable, schema: missions}
  redis: {address: localhost:6379}
support:
  database: {host: localhost, port: "5432", user: postgres, dbName: postgres, sslMode: disable, schema: support}
  redis: {address: localhost:6379}
clips:
  database: {host: localhost, port: "5432", user: postgres, dbName: postgres, sslMode: disable, schema: clips}
auth:
  enabled: true
  enforce: true
  secret: "%s"
  ttl: 2h
  database: {host: localhost, port: "5432", user: postgres, dbName: postgres, sslMode: disable, schema: auth}
  redis: {address: localhost:6379}
`, secret), 0o600))

	return path
}

func TestOnlyAuthReadsTheSeedOutOfTheAuthBlock(t *testing.T) {
	secret, _ := cpsession.TestKeyPair()

	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile(authBlock(t, secret))))

	assert.Equal(t, secret, config.Auth.Secret)
	assert.Equal(t, 2*time.Hour, config.Auth.TTL)
	assert.True(t, config.Planet.Auth.Enabled)
	assert.True(t, config.Planet.Auth.Enforce)

	planetBlock := reflect.TypeOf(config.Planet.Auth)
	for i := range planetBlock.NumField() {
		assert.NotContains(t, strings.ToLower(planetBlock.Field(i).Name), "secret",
			"the planet block must carry nothing that could hold a signing key")
	}
}

func TestASecretThatIsNotASeedIsRefused(t *testing.T) {
	var config Config
	err := cpconfigs.Load(&config, cpconfigs.FromFile(authBlock(t, "not-a-seed")))
	require.ErrorContains(t, err, "auth.secret")
}

func TestTheExampleConfigReachesTheDatabaseBlock(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.Equal(t, "localhost", config.Planet.Database.Host)
	assert.Equal(t, "5432", config.Planet.Database.Port)
	assert.Equal(t, "postgres", config.Planet.Database.DBName)
	assert.Equal(t, "disable", config.Planet.Database.SSLMode)
	assert.Equal(t, "planet", config.Planet.Database.Schema)
	require.NotNil(t, config.Planet.Database.Pool.MaxOpenConns)
	assert.Equal(t, 4, *config.Planet.Database.Pool.MaxOpenConns)
}

func TestTheExampleConfigReachesThePlayerBlock(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.Equal(t, "player", config.Player.Database.Schema)
	require.NoError(t, config.Player.Database.Validate())
}

func TestTheExampleConfigReachesTheMissionsBlock(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.Equal(t, 21*time.Hour, config.Missions.Calendar.Cutoff, "the missions turn when the daily round closes")
	assert.Equal(t, "missions", config.Missions.Database.Schema)
	require.NoError(t, config.Missions.Validate())
}

func TestTheMissionsModuleWithNoDatabaseIsRefused(t *testing.T) {
	assert.ErrorContains(t, Config{}.Validate(), "missions.database")
}

func TestThePlayerModuleWithNoDatabaseIsRefused(t *testing.T) {
	assert.ErrorContains(t, Config{}.Validate(), "player.database")
}

func TestTheExampleConfigReachesTheScopeMultiplier(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.InDelta(t, 10.0, config.Planet.RateLimiter.ScopeMultiplier, 1e-9)
	assert.InDelta(t, 4.0, config.Planet.RateLimiter.V6ScopeMultiplier, 1e-9)
	assert.InDelta(t, 1.0, config.Planet.RateLimiter.GuestScopeMultiplier, 1e-9)
	assert.InDelta(t, 2.0, config.Planet.RateLimiter.LinkedMultiplier, 1e-9)
	require.NotNil(t, config.Planet.RateLimiter.NewAccountClicks)
	assert.InDelta(t, 10.0, *config.Planet.RateLimiter.NewAccountClicks, 1e-9)
	assert.Equal(t, 60, config.Planet.RateLimiter.Burst, "the squashed policy still reads rateLimiter.burst")
}

func TestTheProcessRefusesToStartWithoutADatabase(t *testing.T) {
	config := Config{}
	config.HTTPServer.BindAddress = "0.0.0.0:8080"
	config.Planet.GameMap.MaxIndex = 100

	require.ErrorContains(t, config.Validate(), "database: [host port user dbName sslMode schema] is empty")
}

func TestTheExampleConfigReachesTheChatDatabaseBlock(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.Equal(t, "chat", config.Chat.Database.Schema)
	assert.Equal(t, "localhost", config.Chat.Database.Host)
	assert.Equal(t, 720*time.Hour, config.Chat.Storage.Retention)
	require.NoError(t, config.Chat.Validate())
}

func TestChatWithoutADatabaseIsRefused(t *testing.T) {
	require.ErrorContains(t, Config{}.Validate(), "chat.database: [host port user dbName sslMode schema] is empty")
}

func TestAuthWithoutASecretIsRefused(t *testing.T) {
	config := Config{}
	config.HTTPServer.BindAddress = "0.0.0.0:8080"
	config.Planet.GameMap.MaxIndex = 100
	config.Auth.Enabled = true

	require.ErrorContains(t, config.Validate(), "auth.secret is empty")
}

func TestTheExampleConfigReachesTheAuthBlock(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.False(t, config.Auth.Enabled, "the example must ship without auth")
	assert.Equal(t, "auth", config.Auth.Database.Schema)
	assert.Equal(t, 90*24*time.Hour, config.Auth.Sessions.GuestTTL)
	assert.Equal(t, 24*time.Hour, config.Auth.Sessions.ExtendEvery)
	assert.Equal(t, "session", config.Auth.Turnstile.Action)
	assert.Equal(t, 10, config.Auth.RateLimiter.Burst)
	assert.Equal(t, 30*24*time.Hour, config.Auth.Sessions.LinkedTTL)
	assert.False(t, config.Auth.SignIn.Enabled, "the example must ship without sign-in")
	assert.Equal(t, "http://localhost:5173/auth/callback", config.Auth.SignIn.RedirectURL)
	assert.Equal(t, 90*24*time.Hour, config.Auth.Prune.IdleFor)
	assert.Equal(t, time.Hour, config.Auth.Prune.Interval)
}

func TestTheProviderSecretsComeFromTheEnvironment(t *testing.T) {
	t.Setenv("auth.google.clientSecret", "google-secret")
	t.Setenv("auth.discord.clientSecret", "discord-secret")

	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.Equal(t, "google-secret", config.Auth.Google.ClientSecret)
	assert.Equal(t, "discord-secret", config.Auth.Discord.ClientSecret)
}

func TestAuthWithoutADatabaseIsRefused(t *testing.T) {
	config := Config{}
	config.Auth.Enabled = true
	config.Auth.Secret = "a-secret"

	require.ErrorContains(t, config.Validate(), "auth.database: [host port user dbName sslMode schema] is empty")
}

func TestTheExampleConfigReachesTheSeasonsBlock(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	require.Len(t, config.Seasons.Calendar.List, 1)
	season := config.Seasons.Calendar.List[0]
	assert.Zero(t, season.Number)
	assert.True(t, time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC).Equal(season.EndsAt))
	assert.Equal(t, 2*time.Hour, season.Finale)
	assert.Equal(t, "default", config.Seasons.RuleSets.Days)
	assert.Equal(t, "rush", config.Seasons.RuleSets.Finale)
	assert.Equal(t, "default", config.Seasons.RuleSets.Over)
	assert.Equal(t, "seasons", config.Seasons.Database.Schema)
	require.NoError(t, config.Seasons.Database.Validate())
}

func TestTheSeasonsModuleWithNoDatabaseIsRefused(t *testing.T) {
	assert.ErrorContains(t, Config{}.Validate(), "seasons.database")
}

func TestTheExampleConfigReachesTheSupportBlock(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.Equal(t, "support", config.Support.Database.Schema)
	assert.Equal(t, 90*24*time.Hour, config.Support.Retention)
	assert.Equal(t, time.Hour, config.Support.PruneInterval)
	assert.InDelta(t, 1.0/60, config.Support.RateLimiter.PerSecond, 1e-4, "one a minute")
	assert.Equal(t, 3, config.Support.RateLimiter.Burst)
	require.NoError(t, config.Support.Validate())
}

func TestTheSupportModuleWithNoDatabaseIsRefused(t *testing.T) {
	assert.ErrorContains(t, Config{}.Validate(), "support.database")
}

func TestTheExampleConfigReachesTheClipsBlock(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.Equal(t, "clips", config.Clips.Database.Schema)
	require.NoError(t, config.Clips.Validate())
}

func TestTheClipsModuleWithNoDatabaseIsRefused(t *testing.T) {
	assert.ErrorContains(t, Config{}.Validate(), "clips.database")
}

func TestSeasonsThatDoNotCountUpAreRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
seasons:
  list:
    - number: 1
      endsAt: 2026-10-31T23:00:00Z
      finale: 2h
`), 0o600))

	var config Config
	err := cpconfigs.Load(&config, cpconfigs.FromFile(path))
	require.ErrorContains(t, err, "seasons: list[0].number is 1")
}

func TestTheExampleConfigReachesTheAttachments(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	attachments := config.Support.Attachments
	assert.False(t, attachments.Enabled, "the example has no bucket")
	assert.Equal(t, 10, attachments.MaxPerNote)
	assert.Equal(t, int64(50<<20), attachments.MaxBytes)
	assert.Equal(t, 10*time.Minute, attachments.UploadTTL)
	assert.Equal(t, time.Hour, attachments.LinkTTL)
	assert.Equal(t, "auto", attachments.Bucket.Region)
	assert.Equal(t, "clickplanet-support", attachments.Bucket.Name)
}

func productionConfig(t *testing.T) Config {
	t.Helper()

	const production = "../../../../deploy/vps/backend.yaml"
	file, err := os.ReadFile(production)
	require.NoError(t, err)
	seed, _ := cpsession.TestKeyPair()
	for _, anchor := range regexp.MustCompile(`env://([A-Za-z_][A-Za-z0-9_]*)`).FindAllStringSubmatch(string(file), -1) {
		t.Setenv(anchor[1], "a-secret")
	}
	t.Setenv("SESSION_SECRET", seed)
	t.Setenv("SUPPORT_BUCKET_SECRET", "the-bucket-secret")
	t.Setenv("httpServer.allowedOrigin", "https://clickplanet.lol")

	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile(production)))
	return config
}

func TestTheProductionConfigEnforcesBans(t *testing.T) {
	config := productionConfig(t)

	assert.True(t, config.AntiBot.Enabled)
	assert.True(t, config.AntiBot.Bans.Enforce, "a key the config does not read leaves bans off without a word")
	assert.Equal(t, []time.Duration{24 * time.Hour, 7 * 24 * time.Hour, 3 * 365 * 24 * time.Hour}, config.AntiBot.Bans.BanDurations)
}

func TestTheProductionConfigTakesFilesInItsBucketAndNeedsItsSecret(t *testing.T) {
	config := productionConfig(t)

	attachments := config.Support.Attachments
	assert.True(t, attachments.Enabled)
	assert.Equal(t, "the-bucket-secret", attachments.Bucket.SecretAccessKey)
	assert.Equal(t, "9ec4cd4fdf48006b2944c522219f1eea", attachments.Bucket.AccessKeyID)
	assert.Equal(t, "https://76f6f3883c3a531ae90d1dc4b96bf810.r2.cloudflarestorage.com", attachments.Bucket.Endpoint)
	assert.Equal(t, "clickplanet-support", attachments.Bucket.Name)
	assert.Equal(t, "auto", attachments.Bucket.Region)
	assert.Equal(t, int64(9<<30), attachments.MaxStoredBytes)
	require.NoError(t, config.Support.Validate())

	config.Support.Attachments.Bucket.SecretAccessKey = ""
	require.ErrorContains(t, config.Support.Validate(), "secretAccessKey",
		"a deploy before SUPPORT_BUCKET_SECRET is set refuses the boot")
}

func TestEveryModuleOfProductionTalksToTheSameRedis(t *testing.T) {
	config := productionConfig(t)

	for name, redis := range map[string]cpredis.Config{
		"auth": config.Auth.Redis, "chat": config.Chat.Redis, "player": config.Player.Redis,
		"seasons": config.Seasons.Redis, "support": config.Support.Redis, "missions": config.Missions.Redis,
		"antiBot": config.AntiBot.Redis,
	} {
		assert.Equal(t, config.Planet.Redis, redis, "%s.redis is the planet's block", name)
	}
	assert.Equal(t, "redis:6379", config.Planet.Redis.Address)
	assert.Equal(t, "a-secret", config.Planet.Redis.Password, "from REDIS_PASSWORD")
}

func TestTheAntibotKeepsItsKeysOutsideThePlanetsBlock(t *testing.T) {
	config := productionConfig(t)

	require.True(t, config.AntiBot.Enabled)
	assert.True(t, config.AntiBot.Bans.Enforce)
	assert.Equal(t, "antibot", config.AntiBot.Database.Schema)
	assert.Equal(t, 400*time.Millisecond, config.AntiBot.Metronome.Detector.MaxSpread)
	require.NoError(t, config.AntiBot.Validate())
}

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconfigs"
)

// The shipped file is the schema. Nothing else checks that a key in it still
// reaches the struct it is named after: koanf drops what it cannot match in
// silence, so a renamed field turns a configured bound back into its default
// without a word.
func TestTheExampleConfigStillReachesTheStructs(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	require.True(t, config.Planet.AntiBot.Enabled)

	assert.Equal(t, 30*time.Second, config.Planet.Bonus.Enclose.Duration)
	assert.Equal(t, 3, config.Planet.Bonus.Enclose.Shapes)
	assert.Equal(t, 15, config.Planet.Bonus.Enclose.MaxTiles)

	assert.Equal(t, 72*time.Hour, config.Planet.Ledger.Retention)
	assert.Equal(t, 5*time.Minute, config.Planet.Ledger.SweepInterval)
	assert.Equal(t, time.Second, config.Planet.LedgerStorage.FlushInterval)

	assert.False(t, config.Planet.AntiBot.ShadowBan.Enforce, "the example must ship observing only")
	assert.Equal(t, []time.Duration{24 * time.Hour, 7 * 24 * time.Hour, 3 * 365 * 24 * time.Hour}, config.Planet.AntiBot.ShadowBan.BanDurations)
	assert.Equal(t, time.Minute, config.Planet.AntiBot.ShadowBan.SaveInterval)
	assert.Equal(t, "antibot", config.Planet.AntiBot.Database.Schema)
	assert.Equal(t, 5*time.Minute, config.Planet.AntiBot.ShadowBan.ReflagInterval)

	assert.Equal(t, 2, config.Planet.AntiBot.Jury.MinSuspects)
	assert.Equal(t, 10*time.Minute, config.Planet.AntiBot.Jury.SuspicionWindow)

	assert.Equal(t, time.Minute, config.Planet.AntiBot.Evidence.SaveInterval)
	assert.Equal(t, 72*time.Hour, config.Planet.AntiBot.Evidence.Retention)

	require.True(t, config.Planet.AntiBot.Retaker.Enabled)
	assert.Equal(t, 5*time.Second, config.Planet.AntiBot.Retaker.Detector.ReactionWindow)
	assert.Equal(t, 12, config.Planet.AntiBot.Retaker.Detector.MinReactions)
	assert.Equal(t, 120*time.Millisecond, config.Planet.AntiBot.Retaker.Detector.MaxSpread)
	assert.Equal(t, 250*time.Millisecond, config.Planet.AntiBot.Retaker.Detector.MaxMedian)

	require.True(t, config.Planet.AntiBot.Sequencer.Enabled)
	assert.Equal(t, 40, config.Planet.AntiBot.Sequencer.Detector.MinSteps)
	assert.InDelta(t, 0.75, config.Planet.AntiBot.Sequencer.Detector.MinShare, 1e-9)
	assert.Equal(t, 200, config.Planet.AntiBot.Sequencer.Detector.CertainSteps)
	assert.InDelta(t, 0.95, config.Planet.AntiBot.Sequencer.Detector.CertainShare, 1e-9)

	require.True(t, config.Planet.AntiBot.Metronome.Enabled)
	assert.Equal(t, 3*time.Second, config.Planet.AntiBot.Metronome.Detector.MaxGap)
	assert.Equal(t, 120*time.Millisecond, config.Planet.AntiBot.Metronome.Detector.MaxSpread)
	assert.Equal(t, 120, config.Planet.AntiBot.Metronome.Detector.MinClicks)
	assert.Equal(t, 30*time.Minute, config.Planet.AntiBot.Metronome.Detector.CertainFor)
	assert.Equal(t, 900, config.Planet.AntiBot.Metronome.Detector.CertainClicks)

	require.True(t, config.Planet.AntiBot.Scraper.Enabled)
	assert.InDelta(t, 5, config.Planet.AntiBot.Scraper.Detector.MinMaps, 1e-9)
	assert.InDelta(t, 15, config.Planet.AntiBot.Scraper.Detector.CertainMaps, 1e-9)
	assert.Equal(t, 15*time.Minute, config.Planet.AntiBot.Scraper.Detector.TrackWindow)
}

// The blocks the antibot rewrite did not touch, so that moving one of them is a
// failing test rather than a bound that silently went back to its default.
func TestTheExampleConfigStillCarriesTheRestOfTheFile(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.Equal(t, "0.0.0.0:8080", config.HTTPServer.BindAddress)
	assert.NotZero(t, config.Planet.GameMap.MaxIndex)
	assert.InDelta(t, float64(1), config.Planet.RateLimiter.PerSecond, 1e-9)
	assert.Equal(t, 10, config.Planet.RateLimiter.Burst)
	require.Len(t, config.Planet.Toll.Steps, 3)
	assert.InDelta(t, 0.70, config.Planet.Toll.Steps[2].Share, 1e-9)
	assert.InDelta(t, 3, config.Planet.Toll.Steps[2].Cost, 1e-9)
	require.NoError(t, config.Planet.Validate())
	assert.Equal(t, time.Second, config.Planet.TilesStorage.FlushInterval)
	assert.Equal(t, "127.0.0.1:8081", config.HTTPServer.AdminBindAddress)
	assert.Equal(t, time.Hour, config.Session.TTL)
}

func TestTheExampleConfigReachesTheBombSettings(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.Equal(t, 30*time.Second, config.Planet.Bonus.Bomb.Duration)
	assert.InDelta(t, 10.4, config.Planet.Bonus.Bomb.Rings, 1e-9)
	assert.InDelta(t, 1.0, config.Planet.Bonus.Kinds["bomb"], 1e-9)
	require.NoError(t, config.Planet.Bonus.Validate())
}

func TestBothContextsReadTheSameSessionBlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
httpServer:
  bindAddress: 0.0.0.0:8080
gameMap:
  maxIndex: 100
database: {host: localhost, port: "5432", user: postgres, dbName: postgres, sslMode: disable, schema: planet}
chat:
  database: {host: localhost, port: "5432", user: postgres, dbName: postgres, sslMode: disable, schema: chat}
session:
  enabled: true
  enforce: true
  secret: a-shared-secret
  ttl: 2h
`), 0o600))

	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile(path)))

	assert.Equal(t, config.Session.Config, config.Planet.Session,
		"the mint and the click check derive their signer from one block, so these cannot diverge")
	assert.Equal(t, "a-shared-secret", config.Planet.Session.Secret)
	assert.Equal(t, 2*time.Hour, config.Planet.Session.TTL)
	assert.True(t, config.Planet.Session.Enforce)
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

func TestSessionsWithoutASecretAreRefused(t *testing.T) {
	config := Config{}
	config.HTTPServer.BindAddress = "0.0.0.0:8080"
	config.Planet.GameMap.MaxIndex = 100
	config.Session.Enabled = true

	require.ErrorContains(t, config.Validate(), "session.secret is empty")
}

func TestTheExampleConfigReachesTheAuthAndAccountsBlocks(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.Equal(t, "127.0.0.1:8082", config.HTTPServer.InternalBindAddress)
	assert.False(t, config.Auth.Enabled, "the example must ship without accounts")
	assert.Equal(t, "auth", config.Auth.Database.Schema)
	assert.Equal(t, 90*24*time.Hour, config.Auth.Sessions.GuestTTL)
	assert.Equal(t, 24*time.Hour, config.Auth.Sessions.ExtendEvery)
	assert.False(t, config.Session.Accounts.Enabled)
	assert.Equal(t, 2*time.Second, config.Session.Accounts.Timeout)
}

func TestAuthWithoutADatabaseIsRefused(t *testing.T) {
	config := Config{}
	config.Auth.Enabled = true

	require.ErrorContains(t, config.Validate(), "auth.database: [host port user dbName sslMode schema] is empty")
}

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/configs"
)

// The shipped file is the schema. Nothing else checks that a key in it still
// reaches the struct it is named after: koanf drops what it cannot match in
// silence, so a renamed field turns a configured bound back into its default
// without a word.
func TestTheExampleConfigStillReachesTheStructs(t *testing.T) {
	var config Config
	require.NoError(t, configs.Load(&config, configs.FromFile("example.yaml")))

	require.True(t, config.Clicks.AntiBot.Enabled)

	assert.False(t, config.Clicks.AntiBot.ShadowBan.Enforce, "the example must ship observing only")
	assert.Equal(t, time.Hour, config.Clicks.AntiBot.ShadowBan.BanDuration)
	assert.Equal(t, 5*time.Minute, config.Clicks.AntiBot.ShadowBan.ReflagInterval)

	assert.Equal(t, 2, config.Clicks.AntiBot.Jury.MinSuspects)
	assert.Equal(t, 10*time.Minute, config.Clicks.AntiBot.Jury.SuspicionWindow)

	require.True(t, config.Clicks.AntiBot.Retaker.Enabled)
	assert.Equal(t, 5*time.Second, config.Clicks.AntiBot.Retaker.Detector.ReactionWindow)
	assert.Equal(t, 12, config.Clicks.AntiBot.Retaker.Detector.MinReactions)
	assert.Equal(t, 120*time.Millisecond, config.Clicks.AntiBot.Retaker.Detector.MaxSpread)
	assert.Equal(t, 250*time.Millisecond, config.Clicks.AntiBot.Retaker.Detector.MaxMedian)

	require.True(t, config.Clicks.AntiBot.Sequencer.Enabled)
	assert.Equal(t, 40, config.Clicks.AntiBot.Sequencer.Detector.MinSteps)
	assert.Equal(t, 0.75, config.Clicks.AntiBot.Sequencer.Detector.MinShare)
	assert.Equal(t, 200, config.Clicks.AntiBot.Sequencer.Detector.CertainSteps)
	assert.Equal(t, 0.95, config.Clicks.AntiBot.Sequencer.Detector.CertainShare)

	require.True(t, config.Clicks.AntiBot.Metronome.Enabled)
	assert.Equal(t, 3*time.Second, config.Clicks.AntiBot.Metronome.Detector.MaxGap)
	assert.Equal(t, 120*time.Millisecond, config.Clicks.AntiBot.Metronome.Detector.MaxSpread)
	assert.Equal(t, 120, config.Clicks.AntiBot.Metronome.Detector.MinClicks)
	assert.Equal(t, 30*time.Minute, config.Clicks.AntiBot.Metronome.Detector.CertainFor)
	assert.Equal(t, 900, config.Clicks.AntiBot.Metronome.Detector.CertainClicks)
}

// The blocks the antibot rewrite did not touch, so that moving one of them is a
// failing test rather than a bound that silently went back to its default.
func TestTheExampleConfigStillCarriesTheRestOfTheFile(t *testing.T) {
	var config Config
	require.NoError(t, configs.Load(&config, configs.FromFile("example.yaml")))

	assert.Equal(t, "0.0.0.0:8080", config.HTTPServer.BindAddress)
	assert.NotZero(t, config.Clicks.GameMap.MaxIndex)
	assert.Equal(t, float64(1), config.Clicks.RateLimiter.PerSecond)
	assert.Equal(t, 10, config.Clicks.RateLimiter.Burst)
	assert.Equal(t, 30*time.Second, config.Clicks.TilesStorage.SnapshotInterval)
	assert.Equal(t, time.Hour, config.Session.TTL)
}

func TestBothContextsReadTheSameSessionBlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
httpServer:
  bindAddress: 0.0.0.0:8080
gameMap:
  maxIndex: 100
session:
  enabled: true
  enforce: true
  secret: a-shared-secret
  ttl: 2h
`), 0o600))

	var config Config
	require.NoError(t, configs.Load(&config, configs.FromFile(path)))

	assert.Equal(t, config.Session.Config, config.Clicks.Session,
		"the mint and the click check derive their signer from one block, so these cannot diverge")
	assert.Equal(t, "a-shared-secret", config.Clicks.Session.Secret)
	assert.Equal(t, 2*time.Hour, config.Clicks.Session.TTL)
	assert.True(t, config.Clicks.Session.Enforce)
}

func TestSessionsWithoutASecretAreRefused(t *testing.T) {
	config := Config{}
	config.HTTPServer.BindAddress = "0.0.0.0:8080"
	config.Clicks.GameMap.MaxIndex = 100
	config.Session.Enabled = true

	require.ErrorContains(t, config.Validate(), "session.secret is empty")
}

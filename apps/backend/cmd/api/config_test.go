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

	assert.False(t, config.Planet.AntiBot.ShadowBan.Enforce, "the example must ship observing only")
	assert.Equal(t, time.Hour, config.Planet.AntiBot.ShadowBan.BanDuration)
	assert.Equal(t, 5*time.Minute, config.Planet.AntiBot.ShadowBan.ReflagInterval)

	assert.Equal(t, 2, config.Planet.AntiBot.Jury.MinSuspects)
	assert.Equal(t, 10*time.Minute, config.Planet.AntiBot.Jury.SuspicionWindow)

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
	assert.InDelta(t, 2, config.Planet.Toll.Steps[2].Cost, 1e-9)
	require.NoError(t, config.Planet.Validate())
	assert.Equal(t, 30*time.Second, config.Planet.TilesStorage.SnapshotInterval)
	assert.Equal(t, time.Hour, config.Session.TTL)
}

func TestTheExampleConfigReachesTheBombSettings(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.Equal(t, 30*time.Second, config.Planet.Bonus.BombDuration)
	assert.InDelta(t, 10.4, config.Planet.Bonus.BombRings, 1e-9)
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

func TestSessionsWithoutASecretAreRefused(t *testing.T) {
	config := Config{}
	config.HTTPServer.BindAddress = "0.0.0.0:8080"
	config.Planet.GameMap.MaxIndex = 100
	config.Session.Enabled = true

	require.ErrorContains(t, config.Validate(), "session.secret is empty")
}

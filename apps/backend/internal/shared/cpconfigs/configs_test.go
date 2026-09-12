package cpconfigs_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconfigs"
)

type serverConfig struct {
	BindAddress     string
	StreamHeartbeat time.Duration
}

type testConfig struct {
	HTTPServer serverConfig
	GameMap    struct{ MaxIndex uint32 }
}

type validatedConfig struct {
	err error
}

func (c validatedConfig) Validate() error { return c.err }

func TestTheFileFillsTheStruct(t *testing.T) {
	var config testConfig
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile(write(t, `
httpServer:
  bindAddress: 0.0.0.0:8080
  streamHeartbeat: 30s
gameMap:
  maxIndex: 42
`))))

	assert.Equal(t, "0.0.0.0:8080", config.HTTPServer.BindAddress)
	assert.Equal(t, 30*time.Second, config.HTTPServer.StreamHeartbeat)
	assert.Equal(t, uint32(42), config.GameMap.MaxIndex)
}

func TestTheEnvironmentOverridesTheFile(t *testing.T) {
	t.Setenv("httpServer.bindAddress", "127.0.0.1:9999")

	var config testConfig
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile(write(t, `
httpServer:
  bindAddress: 0.0.0.0:8080
gameMap:
  maxIndex: 42
`))))

	assert.Equal(t, "127.0.0.1:9999", config.HTTPServer.BindAddress)
	assert.Equal(t, uint32(42), config.GameMap.MaxIndex, "the rest of the file still lands")
}

func TestTheEnvironmentAloneIsAWholeConfig(t *testing.T) {
	t.Setenv("httpServer.bindAddress", "127.0.0.1:9999")

	var config testConfig
	require.NoError(t, cpconfigs.Load(&config))

	assert.Equal(t, "127.0.0.1:9999", config.HTTPServer.BindAddress)
}

func TestAMissingFileIsRefused(t *testing.T) {
	var config testConfig
	require.Error(t, cpconfigs.Load(&config, cpconfigs.FromFile("no-such-file.yaml")))
}

func TestAConfigThatRefusesItselfStopsTheLoad(t *testing.T) {
	refusal := errors.New("gameMap.maxIndex is zero")

	config := validatedConfig{err: refusal}
	err := cpconfigs.Load(&config, cpconfigs.FromFile(write(t, "gameMap:\n  maxIndex: 0\n")))

	require.Error(t, err)
	assert.ErrorIs(t, err, cpconfigs.ErrValidation)
	assert.ErrorIs(t, err, refusal, "the config's own reason has to survive")
}

func TestAConfigWithNoValidateIsLoadedAnyway(t *testing.T) {
	var config testConfig
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile(write(t, "gameMap:\n  maxIndex: 1\n"))))
}

func write(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	return path
}

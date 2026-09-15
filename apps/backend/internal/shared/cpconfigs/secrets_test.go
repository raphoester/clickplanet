package cpconfigs_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconfigs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvResolver(t *testing.T) {
	t.Run("resolves a set variable", func(t *testing.T) {
		t.Setenv("CP_TEST_SECRET", "super-secret-value")

		value, err := cpconfigs.EnvResolver{}.Resolve("env://CP_TEST_SECRET")
		require.NoError(t, err)
		assert.Equal(t, "super-secret-value", value)
	})

	t.Run("resolves a set but empty variable, which is not the same as unset", func(t *testing.T) {
		t.Setenv("CP_TEST_EMPTY", "")

		value, err := cpconfigs.EnvResolver{}.Resolve("env://CP_TEST_EMPTY")
		require.NoError(t, err)
		assert.Empty(t, value)
	})

	t.Run("declines another scheme", func(t *testing.T) {
		_, err := cpconfigs.EnvResolver{}.Resolve("awssm://some/secret")
		assert.ErrorIs(t, err, cpconfigs.ErrNotEligible)
	})

	t.Run("declines a plain string", func(t *testing.T) {
		_, err := cpconfigs.EnvResolver{}.Resolve("just-a-value")
		assert.ErrorIs(t, err, cpconfigs.ErrNotEligible)
	})

	t.Run("fails on an unset variable rather than resolving to empty", func(t *testing.T) {
		_, err := cpconfigs.EnvResolver{}.Resolve("env://CP_SURELY_NOT_SET_12345")
		assert.ErrorIs(t, err, cpconfigs.ErrEnvVarNotFound)
	})
}

type anchoredConfig struct {
	Session struct {
		Secret string
		TTL    time.Duration
	}
	Chat struct {
		Service struct {
			TagSalt string
		}
	}
	Plain string
}

func writeConfig(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	return path
}

func TestLoadResolvesAnchors(t *testing.T) {
	const body = `
session:
  secret: env://CP_TEST_SESSION_SECRET
  ttl: 45m
chat:
  service:
    tagSalt: env://CP_TEST_TAG_SALT
plain: not-an-anchor
`

	t.Run("fills the field the anchor sits on", func(t *testing.T) {
		t.Setenv("CP_TEST_SESSION_SECRET", "signing-key")
		t.Setenv("CP_TEST_TAG_SALT", "deadbeef")

		var config anchoredConfig
		require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile(writeConfig(t, body))))

		assert.Equal(t, "signing-key", config.Session.Secret)
		assert.Equal(t, "deadbeef", config.Chat.Service.TagSalt)
	})

	t.Run("leaves a string carrying no scheme alone", func(t *testing.T) {
		t.Setenv("CP_TEST_SESSION_SECRET", "signing-key")
		t.Setenv("CP_TEST_TAG_SALT", "deadbeef")

		var config anchoredConfig
		require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile(writeConfig(t, body))))

		assert.Equal(t, "not-an-anchor", config.Plain)
	})

	// Supplying a DecoderConfig replaces koanf's own hooks, so this is what
	// catches the duration parsing being dropped along with them.
	t.Run("still parses durations", func(t *testing.T) {
		t.Setenv("CP_TEST_SESSION_SECRET", "signing-key")
		t.Setenv("CP_TEST_TAG_SALT", "deadbeef")

		var config anchoredConfig
		require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile(writeConfig(t, body))))

		assert.Equal(t, 45*time.Minute, config.Session.TTL)
	})

	t.Run("refuses the load when an anchored variable is unset", func(t *testing.T) {
		t.Setenv("CP_TEST_TAG_SALT", "deadbeef")
		require.NoError(t, os.Unsetenv("CP_TEST_SESSION_SECRET"))

		var config anchoredConfig
		err := cpconfigs.Load(&config, cpconfigs.FromFile(writeConfig(t, body)))

		require.Error(t, err)
		assert.ErrorIs(t, err, cpconfigs.ErrEnvVarNotFound)
		assert.Contains(t, err.Error(), "CP_TEST_SESSION_SECRET")
	})
}

package log_flags_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/log_flags"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type stubFlags struct{ err error }

func (s stubFlags) Allegiances(context.Context, ...clicks.AllegianceKey) (map[clicks.AllegianceKey]clicks.Allegiance, error) {
	return map[clicks.AllegianceKey]clicks.Allegiance{}, s.err
}

func read(t *testing.T, ctx context.Context, inner stubFlags) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := log_flags.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Allegiances(ctx, clicks.ScopeAllegianceKey("home"))
	return logs.String(), err
}

func TestAReadThatWorksSaysNothing(t *testing.T) {
	logs, err := read(t, t.Context(), stubFlags{})

	require.NoError(t, err)
	assert.Empty(t, logs)
}

func TestAFailedReadIsLoggedUnlessTheProcessIsStopping(t *testing.T) {
	cause := errors.New("postgres is down")

	logs, err := read(t, t.Context(), stubFlags{err: cause})
	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs, "level=ERROR")

	stopped, cancel := context.WithCancel(t.Context())
	cancel()
	logs, _ = read(t, stopped, stubFlags{err: cause})
	assert.Empty(t, logs)
}

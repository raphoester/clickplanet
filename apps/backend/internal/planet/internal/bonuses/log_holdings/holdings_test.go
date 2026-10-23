package log_holdings_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/log_holdings"
)

const alice bonuses.Holder = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"

func logged(store *bonuses.MemoryStore) (*log_holdings.Logged, *bytes.Buffer) {
	var out bytes.Buffer
	return log_holdings.New(store, slog.New(slog.NewTextHandler(&out, nil))), &out
}

func TestAFailedReadIsLoggedWithItsAccount(t *testing.T) {
	store := bonuses.NewMemoryStore()
	down := errors.New("postgres is down")
	store.FailWith(down)
	holdings, out := logged(store)

	_, err := holdings.Held(t.Context(), alice)

	require.ErrorIs(t, err, down)
	assert.Contains(t, out.String(), "account="+string(alice))
	assert.Contains(t, out.String(), "postgres is down")
}

func TestAReadCutByTheShutdownLogsNothing(t *testing.T) {
	store := bonuses.NewMemoryStore()
	store.FailWith(context.Canceled)
	holdings, out := logged(store)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := holdings.Held(ctx, alice)

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, out.String())
}

func TestAReadLogsNothing(t *testing.T) {
	store := bonuses.NewMemoryStore()
	require.NoError(t, bonuses.NewCharges(store, bonuses.ChargesConfig{}).Grant(t.Context(), alice, bonuses.KindBomb, 1))
	holdings, out := logged(store)

	held, err := holdings.Held(t.Context(), alice)

	require.NoError(t, err)
	assert.Equal(t, bonuses.Held{Bomb: true}, held)
	assert.Empty(t, out.String())
}

package log_refusals_test

import (
	"bytes"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/refusals"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/refusals/log_refusals"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestARefusalThatCannotBeReadIsLogged(t *testing.T) {
	var out bytes.Buffer
	store := refusals.NewMemoryStore()
	store.FailWith(errors.New("postgres is down"))
	logged := log_refusals.New(refusals.NewBook(store, cptime.NewFixedClock(time.Now())), slog.New(slog.NewTextHandler(&out, nil)))

	_, refused, err := logged.Refusal(t.Context(), clicks.Payer{Scope: "203.0.113.7"})

	require.Error(t, err)
	assert.False(t, refused)
	assert.Contains(t, out.String(), "postgres is down")
}

func TestAReadThatWorkedLogsNothing(t *testing.T) {
	var out bytes.Buffer
	logged := log_refusals.New(refusals.NewBook(refusals.NewMemoryStore(), cptime.NewFixedClock(time.Now())),
		slog.New(slog.NewTextHandler(&out, nil)))

	_, _, err := logged.Refusal(t.Context(), clicks.Payer{Scope: "203.0.113.7"})

	require.NoError(t, err)
	assert.Empty(t, out.String())
}

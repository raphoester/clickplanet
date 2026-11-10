package log_allegiances_test

import (
	"bytes"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances/log_allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const alice = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"

func logged(store *allegiances.MemoryStore, out *bytes.Buffer) *log_allegiances.Logged {
	book := allegiances.NewBook(store, cpcountries.New(), func() allegiances.Lock { return false },
		cptime.NewFixedClock(time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)), &allegiances.RecordedPledges{})
	return log_allegiances.New(book, slog.New(slog.NewTextHandler(out, nil)))
}

func TestAFlagThatCannotBeReadIsLogged(t *testing.T) {
	var out bytes.Buffer
	store := allegiances.NewMemoryStore()
	store.FailWith(errors.New("postgres is down"))

	err := logged(store, &out).PlayError(t.Context(), clicks.Payer{Account: alice}, "fr")

	require.ErrorIs(t, err, allegiances.ErrUnread)
	assert.Contains(t, out.String(), "postgres is down")
}

func TestARefusedFlagLogsNothing(t *testing.T) {
	var out bytes.Buffer

	err := logged(allegiances.NewMemoryStore(), &out).PlayError(t.Context(), clicks.Payer{Account: alice}, "fr")

	require.ErrorIs(t, err, allegiances.ErrNoFlag)
	assert.Empty(t, out.String())
}

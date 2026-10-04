package log_gift_storage_test

import (
	"bytes"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/gifts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/gifts/inmemory_gift_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/gifts/log_gift_storage"
)

func TestAFailureIsLoggedAndPassedOn(t *testing.T) {
	var logs bytes.Buffer
	store := inmemory_gift_storage.New()
	failure := errors.New("postgres is down")
	store.FailWith(failure)

	err := log_gift_storage.New(store, slog.New(slog.NewTextHandler(&logs, nil))).Give(t.Context(), "finale-0", "ada")

	require.ErrorIs(t, err, failure)
	assert.Contains(t, logs.String(), `level=ERROR msg="failed to give a gift; the account's next click tries again" tag=finale-0 account=ada error="postgres is down"`)
}

func TestAGiftAlreadyGivenIsNoFault(t *testing.T) {
	var logs bytes.Buffer
	store := inmemory_gift_storage.New()
	logged := log_gift_storage.New(store, slog.New(slog.NewTextHandler(&logs, nil)))

	require.NoError(t, logged.Give(t.Context(), "finale-0", "ada"))
	require.ErrorIs(t, logged.Give(t.Context(), "finale-0", "ada"), gifts.ErrGiven)

	assert.Empty(t, logs.String())
}

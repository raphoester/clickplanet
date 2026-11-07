package raids_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids/inmemory_raid_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestNoRaidModeEverSetRunsNot(t *testing.T) {
	_, running, err := raids.NewBook(inmemory_raid_storage.New(), cptime.NewFixedClock(at)).Running(t.Context())

	require.NoError(t, err)
	assert.False(t, running)
}

func TestTheBookSaysARaidModeRunsUntilItEnds(t *testing.T) {
	store, clock := inmemory_raid_storage.New(), cptime.NewFixedClock(at)
	mode, err := raids.Start(at, time.Hour, "")
	require.NoError(t, err)
	require.NoError(t, store.Save(t.Context(), mode))
	book := raids.NewBook(store, clock)

	running, on, err := book.Running(t.Context())
	require.NoError(t, err)
	assert.True(t, on)
	assert.Equal(t, mode, running)

	clock.Advance(time.Hour)
	_, on, err = book.Running(t.Context())
	require.NoError(t, err)
	assert.False(t, on, "it ended by itself")
}

func TestAFailureToReadIsAnError(t *testing.T) {
	store := inmemory_raid_storage.New()
	cause := errors.New("postgres is down")
	store.FailWith(cause)

	_, _, err := raids.NewBook(store, cptime.NewFixedClock(at)).Running(t.Context())

	require.ErrorIs(t, err, cause)
}

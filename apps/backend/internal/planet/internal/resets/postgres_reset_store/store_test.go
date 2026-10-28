package postgres_reset_store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets/postgres_reset_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestAResetIsKeptOnceAndReadByTime(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	store := postgres_reset_store.New(db)
	at := time.Date(2026, 11, 1, 15, 0, 0, 0, time.UTC)
	one, err := resets.New("season-1", at)
	require.NoError(t, err)
	two, err := resets.New("season-2", at.Add(30*24*time.Hour))
	require.NoError(t, err)

	for _, reset := range []resets.Reset{two, one} {
		kept, err := store.Keep(t.Context(), reset)
		require.NoError(t, err)
		assert.True(t, kept)
	}
	kept, err := store.Keep(t.Context(), one)
	require.NoError(t, err)
	assert.False(t, kept, "told again")

	after, err := store.After(t.Context(), at.Add(-time.Second))
	require.NoError(t, err)
	assert.Equal(t, []resets.Reset{one, two}, after)
	after, err = store.After(t.Context(), at)
	require.NoError(t, err)
	assert.Equal(t, []resets.Reset{two}, after)
}

func TestAResetNeedsAnIDAndATime(t *testing.T) {
	_, err := resets.New("", time.Now())
	require.ErrorIs(t, err, resets.ErrInvalidReset)
	_, err = resets.New("season-1", time.Time{})
	require.ErrorIs(t, err, resets.ErrInvalidReset)
}

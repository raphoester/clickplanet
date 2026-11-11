package inmemory_grant_store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/inmemory_grant_store"
)

var at = time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)

func record(t *testing.T, store *inmemory_grant_store.Store, grant charges.GrantID, at time.Time) bool {
	t.Helper()

	first, err := store.Record(t.Context(), grant, at)
	require.NoError(t, err)
	return first
}

func TestAGrantIsRecordedOnce(t *testing.T) {
	store := inmemory_grant_store.New(10 * time.Minute)
	grant, other := charges.GrantID{15: 1}, charges.GrantID{15: 2}

	assert.True(t, record(t, store, grant, at))
	assert.False(t, record(t, store, grant, at.Add(time.Second)), "told twice")
	assert.True(t, record(t, store, other, at))
}

func TestAGrantIsKeptForItsWhileAndThenForgotten(t *testing.T) {
	store := inmemory_grant_store.New(10 * time.Minute)
	grant := charges.GrantID{15: 1}
	record(t, store, grant, at)

	assert.False(t, record(t, store, grant, at.Add(9*time.Minute)), "still kept")
	// The walk runs at most once per keep: the first record after it forgets what is old enough.
	record(t, store, charges.GrantID{15: 2}, at.Add(20*time.Minute))
	assert.True(t, record(t, store, grant, at.Add(20*time.Minute)), "forgotten past its while")
}

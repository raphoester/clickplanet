package postgres_grant_store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/postgres_grant_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestAGrantIsRecordedOnce(t *testing.T) {
	store := postgres_grant_store.New(cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS))
	grant, other := bonuses.GrantID{15: 1}, bonuses.GrantID{15: 2}
	at := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)

	first, err := store.Record(t.Context(), grant, at)
	require.NoError(t, err)
	again, err := store.Record(t.Context(), grant, at.Add(time.Second))
	require.NoError(t, err)
	another, err := store.Record(t.Context(), other, at)
	require.NoError(t, err)

	assert.True(t, first)
	assert.False(t, again, "told twice")
	assert.True(t, another)
}

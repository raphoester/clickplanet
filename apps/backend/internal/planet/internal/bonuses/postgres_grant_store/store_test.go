package postgres_grant_store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/postgres_grant_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
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

func TestAGrantIsReadRecordedOnceItIs(t *testing.T) {
	store := postgres_grant_store.New(cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS))
	grant := bonuses.GrantID{15: 1}

	before, err := store.Recorded(t.Context(), grant)
	require.NoError(t, err)
	_, err = store.Record(t.Context(), grant, time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	after, err := store.Recorded(t.Context(), grant)
	require.NoError(t, err)

	assert.False(t, before)
	assert.True(t, after)
}

func TestAnAdRefillRefusedForTheNetworkKeepsNoGrant(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	noon := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	refills := bonuses.NewAdRefills(postgres_grant_store.New(db), bonuses.TestRules, cptime.NewFixedClock(noon.Add(time.Minute)))
	guest := clicks.Payer{Scope: "203.0.113.7", Account: "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"}
	housemate := clicks.Payer{Scope: "203.0.113.7", Account: "7c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f"}
	record := func(payer clicks.Payer) error {
		return db.InTx(t.Context(), func(ctx context.Context) error { return refills.Record(ctx, payer) })
	}
	require.NoError(t, record(guest))

	require.ErrorIs(t, record(housemate), bonuses.ErrAdRefillUsed)

	housemate.Linked = true
	at, err := refills.AvailableAt(t.Context(), housemate)
	require.NoError(t, err)
	assert.Equal(t, noon, at, "its own grant went back with the refusal")
}

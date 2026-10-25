package charge_fills_query_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills/postgres_fill_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_charge_fills_handler/charge_fills_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var at = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func keep(t *testing.T, store *postgres_fill_store.Store, kept fills.Snapshot) {
	t.Helper()

	fill, err := kept.Restore()
	require.NoError(t, err)
	require.NoError(t, store.Save(t.Context(), fill))
}

func TestTheFillsComeNewestFirstWithHowFarEachGot(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	store := postgres_fill_store.New(db)
	keep(t, store, fills.Snapshot{
		ID: "season-0:finale:fill", Source: fills.SourceInternal, StartedAt: at,
		Reached: "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", Filled: 1200, FinishedAt: at.Add(4 * time.Second),
	})
	keep(t, store, fills.Snapshot{ID: "a1b2", Source: fills.SourceOperator, StartedAt: at.Add(time.Minute), Filled: 500})

	answer, err := charge_fills_query.NewPostgresQuery(db).ChargeFills(t.Context())
	require.NoError(t, err)

	require.Len(t, answer.GetFills(), 2)
	running := answer.GetFills()[0]
	assert.Equal(t, "a1b2", running.GetRequestId())
	assert.Equal(t, planetv1.FillSource_FILL_SOURCE_OPERATOR, running.GetSource())
	assert.True(t, at.Add(time.Minute).Equal(running.GetStartedAt().AsTime()))
	assert.Equal(t, uint32(500), running.GetAccountsFilled())
	assert.Nil(t, running.GetFinishedAt(), "a fill still running has no end")
	done := answer.GetFills()[1]
	assert.Equal(t, "season-0:finale:fill", done.GetRequestId())
	assert.Equal(t, planetv1.FillSource_FILL_SOURCE_INTERNAL, done.GetSource())
	assert.Equal(t, uint32(1200), done.GetAccountsFilled())
	assert.True(t, at.Add(4*time.Second).Equal(done.GetFinishedAt().AsTime()))
}

func TestOnlyTheTwentyNewestAreShown(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	store := postgres_fill_store.New(db)
	for i := range 25 {
		keep(t, store, fills.Snapshot{
			ID: bonuses.FillID(fmt.Sprintf("f%02d", i)), Source: fills.SourceOperator,
			StartedAt: at.Add(time.Duration(i) * time.Minute),
		})
	}

	answer, err := charge_fills_query.NewPostgresQuery(db).ChargeFills(t.Context())
	require.NoError(t, err)

	require.Len(t, answer.GetFills(), 20)
	assert.Equal(t, "f24", answer.GetFills()[0].GetRequestId())
	assert.Equal(t, "f05", answer.GetFills()[19].GetRequestId())
}

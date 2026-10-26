package rule_sets_query_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_rules_handler/rule_sets_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/postgres_rule_set_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestTheSetsComeDefaultFirstThenByNameWithTheirValues(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	store := postgres_rule_set_store.New(db)
	for _, kept := range []rules.Snapshot{
		{Name: "rush", Tempo: 3, Boxes: bonuses.Pace{MinInterval: time.Minute, MaxInterval: 2 * time.Minute, MaxChargesPerHour: 30}},
		{Name: "calm", Tempo: 0.5, Boxes: bonuses.Pace{MinInterval: 10 * time.Minute, MaxInterval: 20 * time.Minute, MaxChargesPerHour: 6}},
	} {
		set, err := kept.Restore()
		require.NoError(t, err)
		require.NoError(t, store.Save(t.Context(), set))
	}

	sets, err := rule_sets_query.NewPostgresQuery(db).Sets(t.Context())
	require.NoError(t, err)

	require.Len(t, sets, 3)
	assert.Equal(t, "default", sets[0].GetName(), "the migration keeps it")
	assert.Equal(t, "calm", sets[1].GetName())
	rush := sets[2]
	assert.Equal(t, "rush", rush.GetName())
	assert.InDelta(t, 3, rush.GetTempo(), 1e-9)
	assert.Equal(t, time.Minute, rush.GetBoxes().GetMinInterval().AsDuration())
	assert.Equal(t, 2*time.Minute, rush.GetBoxes().GetMaxInterval().AsDuration())
	assert.Equal(t, uint32(30), rush.GetBoxes().GetMaxChargesPerHour())
}

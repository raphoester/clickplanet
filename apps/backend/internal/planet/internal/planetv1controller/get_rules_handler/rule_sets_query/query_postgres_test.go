package rule_sets_query_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_rules_handler/rule_sets_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/postgres_rule_set_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/rule_set_message"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestTheSetsComeDefaultFirstThenByNameWithEveryValue(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	store := postgres_rule_set_store.New(db)
	rush := rules.TestSnapshot("rush")
	rush.Tempo = 3
	rush.Bonuses.Boxes = bonuses.Pace{MinInterval: time.Minute, MaxInterval: 2 * time.Minute, MaxChargesPerHour: 30}
	for _, kept := range []rules.Snapshot{rush, rules.TestSnapshot("calm")} {
		set, err := kept.Restore()
		require.NoError(t, err)
		require.NoError(t, store.Save(t.Context(), set))
	}

	sets, err := rule_sets_query.NewPostgresQuery(db).Sets(t.Context())
	require.NoError(t, err)

	require.Len(t, sets, 3)
	assert.Equal(t, "default", sets[0].GetName(), "the migration keeps it")
	assert.Equal(t, "calm", sets[1].GetName())
	assert.True(t, proto.Equal(rule_set_message.Encode(rush), sets[2]), "%v", sets[2])
}

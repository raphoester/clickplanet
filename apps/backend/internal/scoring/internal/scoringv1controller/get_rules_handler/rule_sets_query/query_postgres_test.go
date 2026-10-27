package rule_sets_query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rules/postgres_rule_set_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_rules_handler/rule_sets_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestTheSetsComeDefaultFirstThenByNameWithTheirValues(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "scoring", migrations.FS)
	store := postgres_rule_set_store.New(db)
	rush := rules.Battle("rush", 3)
	for _, kept := range []rules.Snapshot{rush, rules.Battle("calm", 2)} {
		set, err := kept.Restore()
		require.NoError(t, err)
		require.NoError(t, store.Save(t.Context(), set))
	}

	sets, err := rule_sets_query.NewPostgresQuery(db).Sets(t.Context())
	require.NoError(t, err)

	require.Len(t, sets, 3)
	assert.Equal(t, "default", sets[0].GetName(), "the migration keeps it")
	assert.True(t, proto.Equal(rules.Seeded().Rules, sets[0].GetRules()))
	assert.Equal(t, "calm", sets[1].GetName())
	assert.Equal(t, "rush", sets[2].GetName())
	assert.True(t, proto.Equal(rush.Rules, sets[2].GetRules()), "%v", sets[2].GetRules())
}

func TestASetThatCannotPlayIsListedSoItCanBeMended(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "scoring", migrations.FS)
	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`UPDATE rule_sets SET rules = jsonb_set(rules, '{ownRound,times}', '0') WHERE name = 'default'`)
	require.NoError(t, err)

	sets, err := rule_sets_query.NewPostgresQuery(db).Sets(t.Context())
	require.NoError(t, err)

	require.Len(t, sets, 1)
	assert.Equal(t, uint32(0), sets[0].GetRules().GetOwnRound().GetTimes())
}

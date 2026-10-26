package rule_sets_query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/get_rules_handler/rule_sets_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/postgres_rule_set_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestTheSetsComeDefaultFirstThenByNameWithTheirValues(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "antibot", migrations.FS)
	store := postgres_rule_set_store.New(db)
	rush := rules.Named("rush", func(r *antibotv1.Rules) { r.Cohort.CertainMembers = 50 })
	for _, kept := range []rules.Snapshot{rush, rules.Named("calm", rules.AsIs)} {
		require.NoError(t, store.Save(t.Context(), kept))
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

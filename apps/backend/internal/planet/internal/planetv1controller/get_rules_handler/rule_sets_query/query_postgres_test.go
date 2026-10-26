package rule_sets_query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_rules_handler/rule_sets_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/postgres_rule_set_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestTheSetsComeDefaultFirstThenByNameAsKept(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	rush := rules.Named("rush", func(r *planetv1.Rules) { r.Clicks.Tempo = 3 })
	set, err := rush.Restore(parts.All())
	require.NoError(t, err)
	require.NoError(t, postgres_rule_set_store.New(db, parts.All()).Save(t.Context(), set))
	_, err = db.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO rule_sets (name, rules)
		SELECT 'calm', jsonb_set(rules, '{clicks,burst}', '0') FROM rule_sets WHERE name = 'default'`)
	require.NoError(t, err)

	sets, err := rule_sets_query.NewPostgresQuery(db).Sets(t.Context())
	require.NoError(t, err)

	require.Len(t, sets, 3)
	assert.Equal(t, "default", sets[0].GetName(), "the migration keeps it")
	assert.Equal(t, "calm", sets[1].GetName())
	assert.Zero(t, sets[1].GetRules().GetClicks().GetBurst(), "a set that cannot play is listed, so it can be mended")
	assert.Equal(t, "rush", sets[2].GetName())
	assert.True(t, proto.Equal(rush.Rules, sets[2].GetRules()), "%v", sets[2])
}

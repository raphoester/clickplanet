package rule_sets_query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/get_rules_handler/rule_sets_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

func TestTheSetsComeDefaultFirstThenByNameAsKept(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "quizzes", migrations.FS)
	rush := rules.Named("rush", func(r *quizzesv1.Rules) { r.MaxRightAnswersPerHour = proto.Uint32(8) })
	set, err := rush.Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
	require.NoError(t, cprules.NewPostgresStore(db, parts.All()).Save(t.Context(), set))
	_, err = db.On(t.Context()).ExecContext(t.Context(), `
		INSERT INTO rule_sets (name, rules)
		SELECT 'calm', jsonb_set(rules, '{maxRightAnswersPerHour}', '0') FROM rule_sets WHERE name = 'default'`)
	require.NoError(t, err)

	sets, err := rule_sets_query.NewPostgresQuery(db).Sets(t.Context())
	require.NoError(t, err)

	require.Len(t, sets, 3)
	assert.Equal(t, "default", sets[0].GetName(), "the migration keeps it")
	assert.Equal(t, "calm", sets[1].GetName())
	assert.Zero(t, sets[1].GetRules().GetMaxRightAnswersPerHour(), "a set that cannot play is listed, so it can be mended")
	assert.Equal(t, "rush", sets[2].GetName())
	assert.True(t, proto.Equal(rush.Rules, sets[2].GetRules()), "%v", sets[2])
}

package rule_set_names_query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/get_rule_set_names_handler/rule_set_names_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

func TestTheNamesAreEverySetKeptByName(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "quizzes", migrations.FS)
	rush, err := rules.Named("rush", rules.AsIs).Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
	require.NoError(t, cprules.NewPostgresStore(db, parts.All()).Save(t.Context(), rush))

	names, err := rule_set_names_query.NewPostgresQuery(db).Names(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []string{"default", "rush"}, names)
}

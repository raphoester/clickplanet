package rule_set_names_query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rules/postgres_rule_set_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_rule_set_names_handler/rule_set_names_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestTheNamesAreEverySetKeptByName(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "scoring", migrations.FS)
	rush, err := rules.Battle("rush", 3).Restore()
	require.NoError(t, err)
	require.NoError(t, postgres_rule_set_store.New(db).Save(t.Context(), rush))

	names, err := rule_set_names_query.NewPostgresQuery(db).Names(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []string{"default", "rush"}, names)
}

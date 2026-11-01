package rule_set_names_query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/get_rule_set_names_handler/rule_set_names_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/postgres_rule_set_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestTheNamesAreEverySetKeptByName(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "antibot", migrations.FS)
	rush, err := rules.Named("rush", rules.AsIs).Restore(guard.Parts(), rules.Seeded())
	require.NoError(t, err)
	require.NoError(t, postgres_rule_set_store.New(db, guard.Parts()).Save(t.Context(), rush))

	names, err := rule_set_names_query.NewPostgresQuery(db).Names(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []string{"default", "rush"}, names)
}

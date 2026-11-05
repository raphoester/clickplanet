package rule_set_query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_rule_set_handler/rule_set_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/postgres_rule_set_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestASetPlaysItsOwnValuesOverDefaults(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	rush, err := rules.Snapshot{Name: "rush", Rules: &planetv1.Rules{
		Clicks: &planetv1.ClickRules{Tempo: proto.Float64(3)},
	}}.Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
	require.NoError(t, postgres_rule_set_store.New(db, parts.All()).Save(t.Context(), rush))
	query := rule_set_query.NewPostgresQuery(db)

	played, err := query.Played(t.Context(), "rush")

	require.NoError(t, err)
	assert.InDelta(t, 3, played.GetClicks().GetTempo(), 0)
	assert.Equal(t, rules.Seeded().Rules.GetClicks().GetBurst(), played.GetClicks().GetBurst(), "default's, where the set holds none")
	assert.True(t, proto.Equal(rush.TakePlayedSnapshot().Rules, played), "what the set plays")
}

func TestDefaultPlaysAlone(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)

	played, err := rule_set_query.NewPostgresQuery(db).Played(t.Context(), "default")

	require.NoError(t, err)
	assert.True(t, proto.Equal(rules.Seeded().Rules, played))
}

func TestASetNotKeptIsNone(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)

	_, err := rule_set_query.NewPostgresQuery(db).Played(t.Context(), "rush")

	require.ErrorIs(t, err, rule_set_query.ErrNoSet)
}

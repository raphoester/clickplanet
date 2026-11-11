package rule_set_query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/get_rule_set_handler/rule_set_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/rules/postgres_rule_set_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestASetPlaysItsOwnValuesOverDefaults(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "bonuses", migrations.FS)
	rush, err := rules.Snapshot{Name: "rush", Rules: &bonusesv1.Rules{
		Spread: &bonusesv1.SpreadRules{MaxPerBox: proto.Uint32(8)},
	}}.Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
	require.NoError(t, postgres_rule_set_store.New(db, parts.All()).Save(t.Context(), rush))
	query := rule_set_query.NewPostgresQuery(db)

	played, err := query.Played(t.Context(), "rush")

	require.NoError(t, err)
	assert.Equal(t, uint32(8), played.GetSpread().GetMaxPerBox())
	assert.Equal(t, rules.Seeded().Rules.GetEnclose().GetMaxPerBox(), played.GetEnclose().GetMaxPerBox(), "default's, where the set holds none")
	assert.True(t, proto.Equal(rush.TakePlayedSnapshot().Rules, played), "what the set plays")
}

func TestDefaultPlaysAlone(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "bonuses", migrations.FS)

	played, err := rule_set_query.NewPostgresQuery(db).Played(t.Context(), "default")

	require.NoError(t, err)
	assert.True(t, proto.Equal(rules.Seeded().Rules, played))
}

func TestASetNotKeptIsNone(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "bonuses", migrations.FS)

	_, err := rule_set_query.NewPostgresQuery(db).Played(t.Context(), "rush")

	require.ErrorIs(t, err, rule_set_query.ErrNoSet)
}

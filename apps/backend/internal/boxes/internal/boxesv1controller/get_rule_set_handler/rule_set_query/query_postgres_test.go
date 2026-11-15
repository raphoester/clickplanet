package rule_set_query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	boxesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxesv1controller/get_rule_set_handler/rule_set_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

var server cppgtest.Server

func TestMain(m *testing.M) { server.Run(m) }

func TestASetPlaysItsOwnValuesOverDefaults(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "boxes", migrations.FS)
	rush, err := cprules.Snapshot[*boxesv1.Rules]{Name: "rush", Rules: &boxesv1.Rules{
		MaxBoxesPerHour: proto.Uint32(8),
	}}.Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
	require.NoError(t, cprules.NewPostgresStore(db, parts.All()).Save(t.Context(), rush))
	query := rule_set_query.NewPostgresQuery(db)

	played, err := query.Played(t.Context(), "rush")

	require.NoError(t, err)
	assert.Equal(t, uint32(8), played.GetMaxBoxesPerHour())
	assert.Equal(t, rules.Seeded().Rules.GetMissRetry().AsDuration(), played.GetMissRetry().AsDuration(), "default's, where the set holds none")
	assert.True(t, proto.Equal(rush.TakePlayedSnapshot().Rules, played), "what the set plays")
}

func TestDefaultPlaysAlone(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "boxes", migrations.FS)

	played, err := rule_set_query.NewPostgresQuery(db).Played(t.Context(), "default")

	require.NoError(t, err)
	assert.True(t, proto.Equal(rules.Seeded().Rules, played))
}

func TestASetNotKeptIsNone(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "boxes", migrations.FS)

	_, err := rule_set_query.NewPostgresQuery(db).Played(t.Context(), "rush")

	require.ErrorIs(t, err, rule_set_query.ErrNoSet)
}

package rule_set_query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/get_rule_set_handler/rule_set_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

var server cppgtest.Server

func TestMain(m *testing.M) { server.Run(m) }

func TestASetPlaysItsOwnValuesOverDefaults(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "quizzes", migrations.FS)
	rush, err := cprules.Snapshot[*quizzesv1.Rules]{Name: "rush", Rules: &quizzesv1.Rules{
		MaxRightAnswersPerHour: proto.Uint32(8),
	}}.Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
	require.NoError(t, cprules.NewPostgresStore(db, parts.All()).Save(t.Context(), rush))
	query := rule_set_query.NewPostgresQuery(db)

	played, err := query.Played(t.Context(), "rush")

	require.NoError(t, err)
	assert.Equal(t, uint32(8), played.GetMaxRightAnswersPerHour())
	assert.InDelta(t, rules.Seeded().Rules.GetLeaderBias(), played.GetLeaderBias(), 1e-9, "default's, where the set holds none")
	assert.True(t, proto.Equal(rush.TakePlayedSnapshot().Rules, played), "what the set plays")
}

func TestDefaultPlaysAlone(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "quizzes", migrations.FS)

	played, err := rule_set_query.NewPostgresQuery(db).Played(t.Context(), "default")

	require.NoError(t, err)
	assert.True(t, proto.Equal(rules.Seeded().Rules, played))
}

func TestASetNotKeptIsNone(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "quizzes", migrations.FS)

	_, err := rule_set_query.NewPostgresQuery(db).Played(t.Context(), "rush")

	require.ErrorIs(t, err, rule_set_query.ErrNoSet)
}

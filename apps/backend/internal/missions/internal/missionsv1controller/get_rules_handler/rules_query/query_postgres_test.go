package rules_query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/get_rules_handler/rules_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/rules/postgres_rules_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestTheRulesAreReadAsTheStoreKeptThem(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "missions", migrations.FS)
	message := &missionsv1.Rules{FlagMissions: &missionsv1.FlagMissionRules{GoalPercent: 40, LeastGoal: 300, FewestPoints: 2, MostPoints: 8}}
	saved, err := rules.RulesOf(message)
	require.NoError(t, err)
	require.NoError(t, postgres_rules_store.New(db).Save(t.Context(), saved))

	answer, err := rules_query.NewPostgresQuery(db).Rules(t.Context())

	require.NoError(t, err)
	assert.True(t, proto.Equal(&missionsv1.GetRulesResponse{Rules: message}, answer), "%v", answer)
}

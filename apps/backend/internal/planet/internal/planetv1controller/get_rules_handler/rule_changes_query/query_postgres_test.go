package rule_changes_query_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_rules_handler/rule_changes_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/postgres_rule_change_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var at = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func TestTheChangesComeNewestFirstWithTheirSource(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	store := postgres_rule_change_store.New(db)
	require.NoError(t, store.Save(t.Context(), rules.Snapshot{
		Request: "season-0:finale", Set: "rush", Source: rules.SourceInternal, AppliedAt: at,
	}.Restore()))
	require.NoError(t, store.Save(t.Context(), rules.Snapshot{
		Request: "a1b2", Set: rules.Default, Source: rules.SourceOperator, AppliedAt: at.Add(time.Minute),
	}.Restore()))

	changes, err := rule_changes_query.NewPostgresQuery(db).Changes(t.Context())
	require.NoError(t, err)

	require.Len(t, changes, 2)
	assert.Equal(t, "a1b2", changes[0].GetRequestId())
	assert.Equal(t, "default", changes[0].GetRuleSet())
	assert.Equal(t, planetv1.RuleChangeSource_RULE_CHANGE_SOURCE_OPERATOR, changes[0].GetSource())
	assert.True(t, at.Add(time.Minute).Equal(changes[0].GetAppliedAt().AsTime()))
	assert.Equal(t, "season-0:finale", changes[1].GetRequestId())
	assert.Equal(t, planetv1.RuleChangeSource_RULE_CHANGE_SOURCE_INTERNAL, changes[1].GetSource())
}

func TestOnlyTheFiftyNewestAreShown(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	store := postgres_rule_change_store.New(db)
	for i := range 60 {
		require.NoError(t, store.Save(t.Context(), rules.Snapshot{
			Request: rules.RequestID(fmt.Sprintf("r%02d", i)), Set: "rush", Source: rules.SourceOperator,
			AppliedAt: at.Add(time.Duration(i) * time.Minute),
		}.Restore()))
	}

	changes, err := rule_changes_query.NewPostgresQuery(db).Changes(t.Context())
	require.NoError(t, err)

	require.Len(t, changes, 50)
	assert.Equal(t, "r59", changes[0].GetRequestId())
	assert.Equal(t, "r10", changes[49].GetRequestId())
}

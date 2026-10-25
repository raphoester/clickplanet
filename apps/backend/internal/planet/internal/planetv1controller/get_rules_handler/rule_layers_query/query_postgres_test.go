package rule_layers_query_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_rules_handler/rule_layers_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/postgres_rule_layer_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var at = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func TestTheLast20BaselinesComeNewestFirst(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	store := postgres_rule_layer_store.New(db)
	for i := range 25 {
		kept := rules.BaselineSnapshot{
			Request: rules.RequestID(fmt.Sprintf("r%02d", i)), Set: rules.Default, AppliedAt: at.Add(time.Duration(i) * time.Minute),
		}
		require.NoError(t, store.SaveBaseline(t.Context(), kept.Restore()))
	}

	baselines, err := rule_layers_query.NewPostgresQuery(db).Baselines(t.Context())
	require.NoError(t, err)

	require.Len(t, baselines, 20)
	assert.Equal(t, "r24", baselines[0].GetRequestId())
	assert.Equal(t, "default", baselines[0].GetRuleSet())
	assert.True(t, at.Add(24*time.Minute).Equal(baselines[0].GetAppliedAt().AsTime()))
	assert.Equal(t, "r05", baselines[19].GetRequestId())
}

func TestTheOverrideIsTheOneKeptOrNone(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	query := rule_layers_query.NewPostgresQuery(db)

	none, err := query.Override(t.Context())
	require.NoError(t, err)
	assert.Nil(t, none)

	kept := rules.OverrideSnapshot{Set: rules.Default, StartsAt: at, EndsAt: at.Add(time.Hour)}
	require.NoError(t, postgres_rule_layer_store.New(db).SaveOverride(t.Context(), kept.Restore()))

	override, err := query.Override(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "default", override.GetRuleSet())
	assert.True(t, at.Equal(override.GetStartsAt().AsTime()))
	assert.True(t, at.Add(time.Hour).Equal(override.GetEndsAt().AsTime()))
}

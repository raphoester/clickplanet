package postgres_rule_set_store_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/postgres_rule_set_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)

	suite.Run(t, &rules.StoreContractSuite{
		NewStore: func() rules.Store {
			require.NoError(t, db.Purge(t.Context()))
			return postgres_rule_set_store.New(db)
		},
	})
}

func TestTheMigrationSeedsDefaultWithProductionsRules(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)

	kept, err := postgres_rule_set_store.New(db).Set(t.Context(), rules.Default)
	require.NoError(t, err)

	assert.True(t, proto.Equal(rules.Seeded().Rules, kept.Rules), "%v", kept.Rules)
}

func TestAValueThisBuildDoesNotKnowIsIgnored(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`UPDATE rule_sets SET rules = jsonb_set(rules, '{frozen}', 'true') WHERE name = 'default'`)
	require.NoError(t, err)

	kept, err := postgres_rule_set_store.New(db).Set(t.Context(), rules.Default)
	require.NoError(t, err)

	assert.True(t, proto.Equal(rules.Seeded().Rules, kept.Rules))
}

func TestEveryValueIsWrittenZerosIncluded(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	store := postgres_rule_set_store.New(db)
	require.NoError(t, store.Save(t.Context(), rules.Named("calm", rules.AsIs)))

	var written bool
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(),
		`SELECT rules->'bonuses'->'kinds' ? 'refill' AND rules->'bonuses'->'quiz' ? 'leaderBias' FROM rule_sets WHERE name = 'calm'`).
		Scan(&written))
	assert.True(t, written)
}

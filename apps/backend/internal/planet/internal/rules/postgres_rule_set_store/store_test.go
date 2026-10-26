package postgres_rule_set_store_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/postgres_rule_set_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)

	suite.Run(t, &rules.StoreContractSuite{
		NewStore: func() rules.Store {
			require.NoError(t, db.Purge(t.Context()))
			return postgres_rule_set_store.New(db, parts.All())
		},
		Parts: parts.All(),
	})
}

func TestTheMigrationSeedsDefaultWithProductionsRules(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)

	set, err := postgres_rule_set_store.New(db, parts.All()).Set(t.Context(), rules.Default)
	require.NoError(t, err)

	assert.True(t, proto.Equal(rules.Seeded().Rules, set.TakeSnapshot().Rules), "%v", set.TakeSnapshot().Rules)
}

func TestAValueThisBuildDoesNotKnowIsIgnored(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`UPDATE rule_sets SET rules = jsonb_set(rules, '{frozen}', 'true') WHERE name = 'default'`)
	require.NoError(t, err)

	set, err := postgres_rule_set_store.New(db, parts.All()).Set(t.Context(), rules.Default)
	require.NoError(t, err)

	assert.True(t, proto.Equal(rules.Seeded().Rules, set.TakeSnapshot().Rules))
}

func TestASetKeptThatCannotPlayIsRefused(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`UPDATE rule_sets SET rules = jsonb_set(rules, '{clicks,burst}', '0') WHERE name = 'default'`)
	require.NoError(t, err)

	_, err = postgres_rule_set_store.New(db, parts.All()).Set(t.Context(), rules.Default)

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "clicks.burst")
}

func TestEveryValueIsWrittenZerosIncluded(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	calm, err := rules.Named("calm", rules.AsIs).Restore(parts.All())
	require.NoError(t, err)
	require.NoError(t, postgres_rule_set_store.New(db, parts.All()).Save(t.Context(), calm))

	var written bool
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(),
		`SELECT rules->'bonuses'->'kinds' ? 'refill' AND rules->'bonuses'->'quiz' ? 'leaderBias' FROM rule_sets WHERE name = 'calm'`).
		Scan(&written))
	assert.True(t, written)
}

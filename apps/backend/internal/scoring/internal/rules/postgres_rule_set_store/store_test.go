package postgres_rule_set_store_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rules/postgres_rule_set_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "scoring", migrations.FS)

	suite.Run(t, &rules.StoreContractSuite{
		NewStore: func() rules.Store {
			require.NoError(t, db.Purge(t.Context()))
			return postgres_rule_set_store.New(db)
		},
	})
}

func TestTheMigrationSeedsDefaultWithNoRoundOfItsOwn(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "scoring", migrations.FS)

	set, err := postgres_rule_set_store.New(db).Set(t.Context(), rules.Default)
	require.NoError(t, err)

	assert.True(t, proto.Equal(rules.Seeded().Rules, set.TakeSnapshot().Rules), "%v", set.TakeSnapshot().Rules)
}

func TestASetKeptThatCannotPlayIsRefused(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "scoring", migrations.FS)
	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`UPDATE rule_sets SET rules = jsonb_set(rules, '{ownRound,times}', '0') WHERE name = 'default'`)
	require.NoError(t, err)

	_, err = postgres_rule_set_store.New(db).Set(t.Context(), rules.Default)

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "ownRound.times")
}

func TestEveryValueIsWrittenZerosIncluded(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "scoring", migrations.FS)
	calm, err := rules.Snapshot{Name: "calm", Rules: rules.Seeded().Rules}.Restore()
	require.NoError(t, err)
	require.NoError(t, postgres_rule_set_store.New(db).Save(t.Context(), calm))

	var written bool
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(),
		`SELECT rules->'ownRound' ? 'enabled' FROM rule_sets WHERE name = 'calm'`).Scan(&written))
	assert.True(t, written)
}

package postgres_rule_set_store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
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

func TestTheMigrationKeepsTheGamesOwnRulesAsTheDefault(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)

	set, err := postgres_rule_set_store.New(db).Set(t.Context(), rules.Default)
	require.NoError(t, err)

	assert.Equal(t, rules.Snapshot{
		Name:  rules.Default,
		Tempo: 1,
		Boxes: bonuses.Pace{MinInterval: 4 * time.Minute, MaxInterval: 8 * time.Minute, MaxChargesPerHour: 12},
		Crowd: 6,
	}, set.TakeSnapshot(), "production's values on 25 October 2026")
}

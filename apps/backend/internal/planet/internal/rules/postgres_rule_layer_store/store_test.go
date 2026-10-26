package postgres_rule_layer_store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/postgres_rule_layer_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/postgres_rule_set_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)

	suite.Run(t, &rules.LayerStoreContractSuite{
		NewStore: func() rules.LayerStore {
			require.NoError(t, db.Purge(t.Context()))
			keepSets(t, postgres_rule_set_store.New(db), rules.Default, "rush")
			return postgres_rule_layer_store.New(db)
		},
	})
}

func keepSets(t *testing.T, sets *postgres_rule_set_store.Store, names ...rules.SetName) {
	for _, name := range names {
		set, err := rules.Snapshot{
			Name:  name,
			Tempo: 1,
			Boxes: bonuses.Pace{MinInterval: time.Minute, MaxInterval: time.Minute, MaxChargesPerHour: 1},
		}.Restore()
		require.NoError(t, err)
		require.NoError(t, sets.Save(t.Context(), set))
	}
}

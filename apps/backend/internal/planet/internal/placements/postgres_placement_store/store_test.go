package postgres_placement_store_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/placements"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/placements/postgres_placement_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)

	suite.Run(t, &placements.StoreContractSuite{
		NewStore: func() placements.Store {
			require.NoError(t, db.Purge(t.Context()))
			return postgres_placement_store.New(db)
		},
	})
}

package postgres_season_store_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/postgres_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "gamemaster", migrations.FS)

	suite.Run(t, &season.StoreContractSuite{
		NewStore: func() season.Store {
			require.NoError(t, db.Purge(t.Context()))
			return postgres_season_store.New(db)
		},
	})
}

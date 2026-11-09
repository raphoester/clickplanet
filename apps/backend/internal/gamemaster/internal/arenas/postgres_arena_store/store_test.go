package postgres_arena_store_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/postgres_arena_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "gamemaster", migrations.FS)

	suite.Run(t, &arenas.StoreContractSuite{
		NewStore: func() arenas.Store {
			require.NoError(t, db.Purge(t.Context()))
			return postgres_arena_store.New(db)
		},
	})
}

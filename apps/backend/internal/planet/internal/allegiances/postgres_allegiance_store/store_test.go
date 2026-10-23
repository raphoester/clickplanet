package postgres_allegiance_store_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances/postgres_allegiance_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)

	suite.Run(t, &allegiances.StoreContractSuite{
		NewStore: func() allegiances.Store {
			require.NoError(t, db.Purge(t.Context()))
			return postgres_allegiance_store.New(db)
		},
		Account: func(n int) allegiances.Account {
			return allegiances.Account(fmt.Sprintf("01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e%02d", n))
		},
	})
}

func TestAFirstFlagIsKeptWithNoTime(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	store := postgres_allegiance_store.New(db)
	const alice allegiances.Account = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"

	require.NoError(t, store.Change(t.Context(), alice, func(allegiances.Allegiance) (allegiances.Allegiance, error) {
		return allegiances.Snapshot{Flag: "fr"}.Restore(), nil
	}))

	var changedAt *string
	require.NoError(t, db.QueryRowContext(t.Context(), `SELECT changed_at::text FROM allegiances WHERE account = $1`,
		string(alice)).Scan(&changedAt))
	require.Nil(t, changedAt)
}

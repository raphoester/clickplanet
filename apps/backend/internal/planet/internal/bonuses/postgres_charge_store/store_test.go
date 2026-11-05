package postgres_charge_store_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/postgres_charge_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)

	suite.Run(t, &bonuses.StoreContractSuite{
		NewStore: func() bonuses.Store {
			require.NoError(t, db.Purge(t.Context()))
			return postgres_charge_store.New(db)
		},
		Holder: func(n int) bonuses.Holder {
			return bonuses.Holder(fmt.Sprintf("01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e%02d", n))
		},
	})
}

func TestAnEmptyHandIsNoRow(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	store := postgres_charge_store.New(db)
	const alice bonuses.Holder = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"
	rows := func() int {
		var n int
		require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(), `SELECT count(*) FROM charges WHERE account = $1`,
			string(alice)).Scan(&n))
		return n
	}
	keep := func(held bonuses.Held) {
		require.NoError(t, store.Change(t.Context(), alice, func(bonuses.Held) (bonuses.Held, error) {
			return held, nil
		}))
	}

	keep(bonuses.Held{Bomb: true})
	require.Equal(t, 1, rows())

	keep(bonuses.Held{})
	require.Zero(t, rows(), "a hand spent to nothing is deleted rather than kept")
}

package postgres_charge_store_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/postgres_charge_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)

	suite.Run(t, &charges.StoreContractSuite{
		NewStore: func() charges.Store {
			require.NoError(t, db.Purge(t.Context()))
			return postgres_charge_store.New(db)
		},
		Holder: func(n int) charges.Holder {
			return charges.Holder(fmt.Sprintf("01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e%02d", n))
		},
	})
}

func TestAnEmptyHandIsNoRow(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	store := postgres_charge_store.New(db)
	const alice charges.Holder = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"
	rows := func() int {
		var n int
		require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(), `SELECT count(*) FROM charges WHERE account = $1`,
			string(alice)).Scan(&n))
		return n
	}
	keep := func(held charges.Held) {
		require.NoError(t, store.Change(t.Context(), alice, func(charges.Held) (charges.Held, error) {
			return held, nil
		}))
	}

	keep(charges.Held{Bomb: true})
	require.Equal(t, 1, rows())

	keep(charges.Held{})
	require.Zero(t, rows(), "a hand spent to nothing is deleted rather than kept")
}

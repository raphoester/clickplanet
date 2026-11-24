package postgres_payment_store_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments/postgres_payment_store"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "shop", migrations.FS)
	suite.Run(t, &payments.StoreContractSuite{
		NewStore: func() payments.Store {
			require.NoError(t, db.Purge(t.Context()))
			return postgres_payment_store.New(db)
		},
	})
}

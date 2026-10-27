package postgres_conviction_store_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/convictions"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/postgres_conviction_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "antibot", migrations.FS)

	suite.Run(t, &convictions.StoreContractSuite{
		NewStore: func() convictions.Store {
			require.NoError(t, db.Purge(t.Context()))
			return postgres_conviction_store.New(db)
		},
		Account: func(n int) string { return fmt.Sprintf("0b7e5b6c-8f3a-4d2e-9c1a-%012x", n) },
	})
}

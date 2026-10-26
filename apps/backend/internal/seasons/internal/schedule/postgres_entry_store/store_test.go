package postgres_entry_store_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/schedule/postgres_entry_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "seasons", migrations.FS)

	suite.Run(t, &schedule.StoreContractSuite{
		NewStore: func() schedule.Store {
			require.NoError(t, db.Purge(t.Context()))
			return postgres_entry_store.New(db)
		},
	})
}

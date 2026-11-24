package postgres_letter_store_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters/postgres_letter_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/migrations"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "support", migrations.FS)
	suite.Run(t, &letters.StorageContractSuite{
		NewStorage: func() letters.Storage {
			require.NoError(t, db.Purge(t.Context()))
			return postgres_letter_store.New(db)
		},
	})
}

package postgres_submission_store_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/postgres_submission_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "flags", migrations.FS)
	suite.Run(t, &submissions.StoreContractSuite{
		NewStore: func() submissions.Store {
			require.NoError(t, db.Purge(t.Context()))
			return postgres_submission_store.New(db)
		},
	})
}

package postgres_appeal_store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/appeals"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/appeals/postgres_appeal_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/postgres_submission_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "flags", migrations.FS)
	submitted := postgres_submission_store.New(db)
	suite.Run(t, &appeals.StoreContractSuite{
		NewStore: func() appeals.Store {
			require.NoError(t, db.Purge(t.Context()))
			return postgres_appeal_store.New(db)
		},
		Seed: func(ctx context.Context, submission submissions.Snapshot) error {
			return submitted.Save(ctx, submissions.Restored(submission))
		},
	})
}

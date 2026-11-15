package postgres_notice_store_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/postgres_notice_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestTheStoreKeepsTheContract(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "antibot", migrations.FS)

	suite.Run(t, &notices.StoreContractSuite{NewStore: func() notices.Store {
		require.NoError(t, db.Purge(t.Context()))
		return postgres_notice_store.New(db)
	}})
}

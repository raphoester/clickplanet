package postgres_unlock_store_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/unlocks"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/unlocks/postgres_unlock_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestRunSuite(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "bonuses", migrations.FS)

	suite.Run(t, &unlocks.StoreContractSuite{NewStore: func() unlocks.Store {
		require.NoError(t, db.Purge(t.Context()))
		return postgres_unlock_store.New(db)
	}})
}

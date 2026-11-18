package postgres_era_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras/postgres_era_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

var server cppgtest.Server

func TestMain(m *testing.M) { server.Run(m) }

// Each test opens a database of its own: the suite reads the rows the migration seeds.
func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &eras.StoreContractSuite{NewStore: func() eras.Store {
		return postgres_era_store.New(server.CreateDatabase(t).OpenSchema(t, "planet", migrations.FS))
	}})
}

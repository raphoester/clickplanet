package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

var server cppgtest.Server

func TestMain(m *testing.M) { server.Run(m) }

func TestThePostgresChoicesKeepTheContract(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "planet", migrations.FS)

	suite.Run(t, &cprules.ChoiceStoreContractSuite{
		NewStore: func() cprules.Choices {
			require.NoError(t, db.Purge(t.Context()))
			return cprules.NewPostgresChoices(db)
		},
	})
}

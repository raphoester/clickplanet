package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

func TestThePostgresChoicesKeepTheContract(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "bonuses", migrations.FS)

	suite.Run(t, &cprules.ChoiceStoreContractSuite{
		NewStore: func() cprules.Choices {
			require.NoError(t, db.Purge(t.Context()))
			return cprules.NewPostgresChoices(db)
		},
	})
}

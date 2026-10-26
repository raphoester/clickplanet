package postgres_choice_store_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/postgres_choice_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "antibot", migrations.FS)

	suite.Run(t, &rules.ChoiceStoreContractSuite{
		NewStore: func() rules.Choices {
			require.NoError(t, db.Purge(t.Context()))
			return postgres_choice_store.New(db)
		},
	})
}

package postgres_contribution_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/postgres_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	standings.StoreContractSuite

	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "seasons", migrations.FS)
	store := postgres_contribution_store.New(s.db)
	s.NewStore = func() standings.Store { return store }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StoreContractSuite.SetupTest()
}

func (s *testSuite) TestOneRowIsMainPerAccountAndSeason() {
	for _, country := range []standings.Country{"fr", "de", "de", "it", "it", "it"} {
		s.Require().NoError(postgres_contribution_store.New(s.db).RecordTake(s.T().Context(), 0,
			standings.Take{Account: standings.AccountID{15: 1}, Country: country}))
	}

	var mains, rows int
	s.Require().NoError(s.db.QueryRowContext(s.T().Context(),
		`SELECT count(*) FILTER (WHERE main), count(*) FROM contributions`).Scan(&mains, &rows))
	s.Equal(1, mains)
	s.Equal(3, rows)
}

package postgres_judgement_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements/postgres_judgement_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	judgements.StorageContractSuite

	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "chat", migrations.FS)
	store := postgres_judgement_store.New(s.db)
	s.NewStorage = func() judgements.Storage { return store }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StorageContractSuite.SetupTest()
}

func (s *testSuite) TestEveryVerdictIsAValueOfThePostgresType() {
	for _, verdict := range []judgements.Verdict{judgements.Acted, judgements.Dismissed, judgements.LeftOpen} {
		_, err := s.db.On(s.T().Context()).ExecContext(s.T().Context(), `SELECT $1::judgement_verdict`, verdict.String())
		s.Require().NoError(err, verdict.String())
	}
}

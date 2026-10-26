package postgres_rule_set_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/postgres_rule_set_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite
	server *cppg.TestServer
	db     *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.server = cppg.StartTestServer(s.T())
	s.db = s.server.OpenSchema(s.T(), "antibot", migrations.FS)
}

func (s *testSuite) TestItKeepsTheContract() {
	suite.Run(s.T(), &rules.StoreContractSuite{
		NewStore: func() rules.Store {
			s.Require().NoError(s.db.Purge(s.T().Context()))
			return postgres_rule_set_store.New(s.db)
		},
	})
}

func (s *testSuite) TestTheMigrationSeedsDefaultWithProductionsRules() {
	seeded := s.server.OpenSchema(s.T(), "antibot_seeded", migrations.FS)

	set, err := postgres_rule_set_store.New(seeded).Set(s.T().Context(), rules.Default)

	s.Require().NoError(err)
	s.Equal(rules.Seeded(), set.TakeSnapshot())
}

func (s *testSuite) TestAKeptDocumentThatCannotPlayIsRefused() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	_, err := s.db.On(s.T().Context()).ExecContext(s.T().Context(), `INSERT INTO rule_sets (name, rules) VALUES ('broken', '{"jury": {}}')`)
	s.Require().NoError(err)

	_, err = postgres_rule_set_store.New(s.db).Set(s.T().Context(), "broken")

	s.Require().ErrorIs(err, rules.ErrInvalidSet)
	s.ErrorContains(err, "jury.minSuspects is 0")
}

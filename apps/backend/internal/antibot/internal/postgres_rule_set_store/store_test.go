package postgres_rule_set_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

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

	kept, err := postgres_rule_set_store.New(seeded).Set(s.T().Context(), rules.Default)

	s.Require().NoError(err)
	s.True(proto.Equal(rules.Seeded().Rules, kept.Rules), "%v", kept.Rules)
}

func (s *testSuite) TestAKeyTheProtoNoLongerHasIsLeftAlone() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	_, err := s.db.On(s.T().Context()).ExecContext(s.T().Context(),
		`INSERT INTO rule_sets (name, rules) VALUES ('old', '{"jury": {"minSuspects": 3, "retired": 1}}')`)
	s.Require().NoError(err)

	kept, err := postgres_rule_set_store.New(s.db).Set(s.T().Context(), "old")

	s.Require().NoError(err)
	s.Equal(uint32(3), kept.Rules.GetJury().GetMinSuspects())
}

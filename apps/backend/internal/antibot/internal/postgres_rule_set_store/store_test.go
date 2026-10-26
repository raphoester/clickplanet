package postgres_rule_set_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
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
		Parts: guard.Parts(),
		NewStore: func() rules.Store {
			s.Require().NoError(s.db.Purge(s.T().Context()))
			return postgres_rule_set_store.New(s.db, guard.Parts())
		},
	})
}

func (s *testSuite) TestTheMigrationSeedsDefaultWithProductionsRules() {
	seeded := s.server.OpenSchema(s.T(), "antibot_seeded", migrations.FS)

	set, err := postgres_rule_set_store.New(seeded, guard.Parts()).Set(s.T().Context(), rules.Default)

	s.Require().NoError(err)
	kept := set.TakeSnapshot()
	s.True(proto.Equal(rules.Seeded().Rules, kept.Rules), "%v", kept.Rules)
}

func (s *testSuite) keepOld() *postgres_rule_set_store.Store {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	store := postgres_rule_set_store.New(s.db, guard.Parts())
	set, err := rules.Named("old", func(r *antibotv1.Rules) { r.Jury.MinSuspects = 3 }).Restore(guard.Parts())
	s.Require().NoError(err)
	s.Require().NoError(store.Save(s.T().Context(), set))
	return store
}

func (s *testSuite) editByHand(query string) {
	_, err := s.db.On(s.T().Context()).ExecContext(s.T().Context(), query)
	s.Require().NoError(err)
}

func (s *testSuite) TestAKeyTheProtoNoLongerHasIsLeftAlone() {
	store := s.keepOld()
	s.editByHand(`UPDATE rule_sets SET rules = jsonb_set(rules, '{jury,retired}', '1') WHERE name = 'old'`)

	set, err := store.Set(s.T().Context(), "old")

	s.Require().NoError(err)
	s.Equal(uint32(3), set.TakeSnapshot().Rules.GetJury().GetMinSuspects())
}

func (s *testSuite) TestAKeptSetThatCannotPlayIsRefused() {
	store := s.keepOld()
	s.editByHand(`UPDATE rule_sets SET rules = jsonb_set(rules, '{jury,minSuspects}', '0') WHERE name = 'old'`)

	_, err := store.Set(s.T().Context(), "old")

	s.Require().ErrorIs(err, rules.ErrInvalidSet)
	s.ErrorContains(err, `"old"`)
	s.ErrorContains(err, "jury.minSuspects")
}

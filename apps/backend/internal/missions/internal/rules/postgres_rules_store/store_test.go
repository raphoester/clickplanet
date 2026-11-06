package postgres_rules_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/rules/postgres_rules_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db    *cppg.Postgres
	store *postgres_rules_store.Store
}

func (s *testSuite) SetupTest() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "missions", migrations.FS)
	s.store = postgres_rules_store.New(s.db)
}

func (s *testSuite) TestTheMigrationSeedsTheNumbersTheCodeHeld() {
	sizing, err := s.store.Sizing(s.T().Context())

	s.Require().NoError(err)
	s.Equal(flagmissions.TestingSizing(), sizing)
}

func (s *testSuite) TestRulesSavedAreReadBack() {
	saved, err := rules.RulesOf(&missionsv1.Rules{FlagMissions: &missionsv1.FlagMissionRules{
		GoalPercent: 40, LeastGoal: 300, FewestPoints: 2, MostPoints: 8,
	}})
	s.Require().NoError(err)

	s.Require().NoError(s.store.Save(s.T().Context(), saved))

	read, err := s.store.Rules(s.T().Context())
	s.Require().NoError(err)
	s.Equal(saved, read)
	sizing, err := s.store.Sizing(s.T().Context())
	s.Require().NoError(err)
	s.Equal(saved.FlagMissions(), sizing)
}

func (s *testSuite) TestRulesKeptThatCannotPlayAreNoDraw() {
	_, err := s.db.On(s.T().Context()).ExecContext(s.T().Context(),
		`UPDATE rules SET rules = '{"flagMissions": {"goalPercent": 0, "leastGoal": 250, "fewestPoints": 3, "mostPoints": 10}}'`)
	s.Require().NoError(err)

	_, err = s.store.Sizing(s.T().Context())

	s.ErrorIs(err, rules.ErrInvalidRules)
}

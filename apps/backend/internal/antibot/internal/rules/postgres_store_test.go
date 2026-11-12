package rules_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite
	database cppgtest.Database
	db       *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.database = server.CreateDatabase(s.T())
	s.db = s.database.OpenSchema(s.T(), "antibot", migrations.FS)
}

func (s *testSuite) TestItKeepsTheContract() {
	suite.Run(s.T(), &cprules.StoreContractSuite[*antibotv1.Rules]{
		Parts:   guard.Parts(),
		Default: rules.Seeded(),
		Rush: func() *antibotv1.Rules {
			return &antibotv1.Rules{
				Cohort:    &antibotv1.CohortRules{CertainMembers: proto.Uint32(50)},
				Metronome: &antibotv1.MetronomeRules{Clock: &antibotv1.ClockRules{MinCoherence: proto.Float64(0.9)}},
				Bans:      &antibotv1.BanRules{Enforce: proto.Bool(false)},
			}
		},
		Other: func() *antibotv1.Rules {
			return &antibotv1.Rules{Jury: &antibotv1.JuryRules{MinSuspects: proto.Uint32(3)}}
		},
		NewStore: func() cprules.Store[*antibotv1.Rules] {
			s.Require().NoError(s.db.Purge(s.T().Context()))
			return cprules.NewPostgresStore(s.db, guard.Parts())
		},
	})
}

func (s *testSuite) TestTheMigrationSeedsDefaultWithProductionsRules() {
	seeded := s.database.OpenSchema(s.T(), "antibot_seeded", migrations.FS)

	set, err := cprules.NewPostgresStore(seeded, guard.Parts()).Set(s.T().Context(), cprules.Default)

	s.Require().NoError(err)
	kept := set.TakeSnapshot()
	s.True(proto.Equal(rules.Seeded().Rules, kept.Rules), "%v", kept.Rules)
}

func (s *testSuite) keepOld() *cprules.PostgresStore[*antibotv1.Rules] {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	store := cprules.NewPostgresStore(s.db, guard.Parts())
	defaults, err := rules.Seeded().Restore(guard.Parts(), rules.Seeded())
	s.Require().NoError(err)
	s.Require().NoError(store.Save(s.T().Context(), defaults))
	set, err := rules.Named("old", func(r *antibotv1.Rules) { r.Jury.MinSuspects = proto.Uint32(3) }).Restore(guard.Parts(), rules.Seeded())
	s.Require().NoError(err)
	s.Require().NoError(store.Save(s.T().Context(), set))
	return store
}

func (s *testSuite) editByHand(query string) {
	_, err := s.db.On(s.T().Context()).ExecContext(s.T().Context(), query)
	s.Require().NoError(err)
}

func (s *testSuite) TestAKeptSetThatCannotPlayIsRefused() {
	store := s.keepOld()
	s.editByHand(`UPDATE rule_sets SET rules = jsonb_set(rules, '{jury,minSuspects}', '0') WHERE name = 'old'`)

	_, err := store.Set(s.T().Context(), "old")

	s.Require().ErrorIs(err, cprules.ErrInvalidSet)
	s.ErrorContains(err, `"old"`)
	s.ErrorContains(err, "jury.minSuspects")
}

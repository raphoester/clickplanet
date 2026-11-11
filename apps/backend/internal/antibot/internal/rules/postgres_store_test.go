package rules_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
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
	seeded := s.server.OpenSchema(s.T(), "antibot_seeded", migrations.FS)

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

func (s *testSuite) TestAKeyTheProtoNoLongerHasIsLeftAlone() {
	store := s.keepOld()
	s.editByHand(`UPDATE rule_sets SET rules = jsonb_set(rules, '{jury,retired}', '1') WHERE name = 'old'`)

	set, err := store.Set(s.T().Context(), "old")

	s.Require().NoError(err)
	s.Equal(uint32(3), set.TakeSnapshot().Rules.GetJury().GetMinSuspects())
}

func (s *testSuite) TestASetIsWrittenWithTheValuesItHoldsZerosIncluded() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	store := cprules.NewPostgresStore(s.db, guard.Parts())
	off, err := cprules.Snapshot[*antibotv1.Rules]{Name: "off", Rules: &antibotv1.Rules{
		Bans: &antibotv1.BanRules{Enforce: proto.Bool(false)},
	}}.Restore(guard.Parts(), rules.Seeded())
	s.Require().NoError(err)
	s.Require().NoError(store.Save(s.T().Context(), off))

	s.JSONEq(`{"bans": {"enforce": false}}`, s.document("off"))
}

const inheritDefault = "20261031180000"

func (s *testSuite) TestTheMigrationKeepsInEachSetWhatItChangesAndItJudgesAsBefore() {
	db := s.server.OpenSchema(s.T(), "antibot_inherit", cppg.MigrationsBefore(s.T(), migrations.FS, inheritDefault))
	s.exec(db, `
		INSERT INTO rule_sets (name, rules)
		SELECT 'rush', jsonb_set(jsonb_set(rules, '{cohort,certainMembers}', '50'), '{metronome,stamina,minBusy}', 'null')
		FROM rule_sets WHERE name = 'default'`)

	s.Require().NoError(db.Migrate(s.T().Context(), migrations.FS))

	var kept string
	s.Require().NoError(db.On(s.T().Context()).QueryRowContext(s.T().Context(),
		`SELECT rules FROM rule_sets WHERE name = 'rush'`).Scan(&kept))
	s.JSONEq(`{"cohort": {"certainMembers": 50}, "metronome": {"stamina": {"minBusy": "0s"}}}`, kept)

	rush, err := cprules.NewPostgresStore(db, guard.Parts()).Set(s.T().Context(), "rush")
	s.Require().NoError(err)
	before := rules.Named("rush", func(r *antibotv1.Rules) {
		r.Cohort.CertainMembers = proto.Uint32(50)
		r.Metronome.Stamina.MinBusy = durationpb.New(0)
	})
	s.True(proto.Equal(before.Rules, rush.TakePlayedSnapshot().Rules), "%v", rush.TakePlayedSnapshot().Rules)
}

func (s *testSuite) TestTheMigrationRefusesASetThatLeavesUnsetABoundDefaultHolds() {
	db := s.server.OpenSchema(s.T(), "antibot_unset", cppg.MigrationsBefore(s.T(), migrations.FS, inheritDefault))
	s.exec(db, `
		UPDATE rule_sets SET rules = jsonb_set(rules, '{metronome,clock,minCoherence}', '0.9') WHERE name = 'default';
		INSERT INTO rule_sets (name, rules)
		SELECT 'rush', rules #- '{metronome,clock,minCoherence}' FROM rule_sets WHERE name = 'default'`)

	err := db.Migrate(s.T().Context(), migrations.FS)

	s.Require().Error(err)
	s.ErrorContains(err, "minCoherence")
}

func (s *testSuite) exec(db *cppg.Postgres, query string) {
	_, err := db.On(s.T().Context()).ExecContext(s.T().Context(), query)
	s.Require().NoError(err)
}

func (s *testSuite) document(name string) string {
	var kept string
	s.Require().NoError(s.db.On(s.T().Context()).QueryRowContext(s.T().Context(),
		`SELECT rules FROM rule_sets WHERE name = $1`, name).Scan(&kept))
	return kept
}

func (s *testSuite) TestAKeptSetThatCannotPlayIsRefused() {
	store := s.keepOld()
	s.editByHand(`UPDATE rule_sets SET rules = jsonb_set(rules, '{jury,minSuspects}', '0') WHERE name = 'old'`)

	_, err := store.Set(s.T().Context(), "old")

	s.Require().ErrorIs(err, cprules.ErrInvalidSet)
	s.ErrorContains(err, `"old"`)
	s.ErrorContains(err, "jury.minSuspects")
}

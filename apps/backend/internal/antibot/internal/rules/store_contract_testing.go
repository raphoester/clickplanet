//go:build testing

package rules

import (
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
)

type StoreContractSuite struct {
	suite.Suite

	Parts    Parts
	NewStore func() Store

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
	s.save(Seeded())
}

func (s *StoreContractSuite) save(kept Snapshot) {
	set, err := kept.Restore(s.Parts, Seeded())
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), set))
}

func (s *StoreContractSuite) set(name SetName) Set {
	set, err := s.store.Set(s.T().Context(), name)
	s.Require().NoError(err)
	return set
}

func (s *StoreContractSuite) requireKept(want Snapshot) {
	kept := s.set(want.Name).TakeSnapshot()
	s.Equal(want.Name, kept.Name)
	s.True(proto.Equal(want.Rules, kept.Rules), "%v", kept.Rules)
}

func rush() Snapshot {
	return Snapshot{Name: "rush", Rules: &antibotv1.Rules{
		Cohort:    &antibotv1.CohortRules{CertainMembers: proto.Uint32(50)},
		Metronome: &antibotv1.MetronomeRules{Clock: &antibotv1.ClockRules{MinCoherence: proto.Float64(0.9)}},
		Bans:      &antibotv1.BanRules{Enforce: proto.Bool(false)},
	}}
}

func (s *StoreContractSuite) TestASetIsReadAsItWasKept() {
	s.save(rush())

	s.requireKept(rush())
}

func (s *StoreContractSuite) TestASetJudgesWithDefaultsValuesForThoseItLeavesUnset() {
	s.save(rush())

	played := Named("rush", func(r *antibotv1.Rules) {
		r.Cohort.CertainMembers = proto.Uint32(50)
		r.Metronome.Clock.MinCoherence = proto.Float64(0.9)
		r.Bans.Enforce = proto.Bool(false)
	})
	s.True(proto.Equal(played.Rules, s.set("rush").TakePlayedSnapshot().Rules))
}

func (s *StoreContractSuite) TestASetFollowsDefaultWhereItHoldsNothing() {
	s.save(rush())
	s.save(Named(Default, func(r *antibotv1.Rules) { r.Jury.MinSuspects = proto.Uint32(3) }))

	played := s.set("rush").TakePlayedSnapshot().Rules
	s.Equal(uint32(50), played.GetCohort().GetCertainMembers())
	s.Equal(uint32(3), played.GetJury().GetMinSuspects())
}

func (s *StoreContractSuite) TestASetNeverKeptIsUnknown() {
	_, err := s.store.Set(s.T().Context(), "frozen")
	s.Require().ErrorIs(err, ErrUnknownSet)
}

func (s *StoreContractSuite) TestASetKeptAgainReplacesItsValues() {
	s.save(rush())
	calmer := Snapshot{Name: "rush", Rules: &antibotv1.Rules{Cohort: &antibotv1.CohortRules{CertainMembers: proto.Uint32(20)}}}
	s.save(calmer)

	s.requireKept(calmer)
}

func (s *StoreContractSuite) TestNamesAreEverySetKept() {
	s.save(Snapshot{Name: "rush", Rules: &antibotv1.Rules{}})

	names, err := s.store.Names(s.T().Context())
	s.Require().NoError(err)
	s.Equal([]SetName{Default, "rush"}, names)
}

//go:build testing

package rules

import (
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store
	Parts    Parts

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

func (s *StoreContractSuite) TestASetIsReadAsItWasKept() {
	rush := Snapshot{Name: "rush", Rules: &planetv1.Rules{Clicks: &planetv1.ClickRules{Tempo: proto.Float64(3)}}}
	s.save(rush)

	kept := s.set("rush").TakeSnapshot()
	s.Equal(SetName("rush"), kept.Name)
	s.True(proto.Equal(rush.Rules, kept.Rules), "%v", kept.Rules)
}

func (s *StoreContractSuite) TestASetPlaysDefaultsValuesForThoseItLeavesUnset() {
	s.save(Snapshot{Name: "rush", Rules: &planetv1.Rules{
		Clicks:  &planetv1.ClickRules{Tempo: proto.Float64(3)},
		Bonuses: &planetv1.BonusRules{Quiz: &planetv1.QuizRules{Enabled: proto.Bool(false)}},
	}})

	played := Named("rush", func(r *planetv1.Rules) {
		r.Clicks.Tempo = proto.Float64(3)
		r.Bonuses.Quiz.Enabled = proto.Bool(false)
	})
	s.True(proto.Equal(played.Rules, s.set("rush").TakePlayedSnapshot().Rules))
}

func (s *StoreContractSuite) TestASetFollowsDefaultWhereItHoldsNothing() {
	s.save(Snapshot{Name: "rush", Rules: &planetv1.Rules{Clicks: &planetv1.ClickRules{Tempo: proto.Float64(3)}}})
	s.save(Named(Default, func(r *planetv1.Rules) { r.Clicks.Burst = proto.Uint32(40) }))

	played := s.set("rush").TakePlayedSnapshot().Rules
	s.InDelta(3.0, played.GetClicks().GetTempo(), 1e-9)
	s.Equal(uint32(40), played.GetClicks().GetBurst())
}

func (s *StoreContractSuite) TestASetThatHoldsATollWithNoStepPlaysNone() {
	s.save(Snapshot{Name: "free", Rules: &planetv1.Rules{Toll: &planetv1.TollRules{}}})

	s.Empty(s.set("free").TakePlayedSnapshot().Rules.GetToll().GetSteps())
}

func (s *StoreContractSuite) TestASetNeverKeptIsUnknown() {
	_, err := s.store.Set(s.T().Context(), "frozen")
	s.Require().ErrorIs(err, ErrUnknownSet)
}

func (s *StoreContractSuite) TestASetKeptAgainReplacesItsValues() {
	s.save(Snapshot{Name: "rush", Rules: &planetv1.Rules{Clicks: &planetv1.ClickRules{Tempo: proto.Float64(3)}}})
	quicker := Snapshot{Name: "rush", Rules: &planetv1.Rules{
		Bonuses: &planetv1.BonusRules{Shield: &planetv1.ShieldRules{PerTile: proto.Uint32(20)}},
	}}
	s.save(quicker)

	s.True(proto.Equal(quicker.Rules, s.set("rush").TakeSnapshot().Rules))
}

func (s *StoreContractSuite) TestNamesAreEverySetKept() {
	s.save(Snapshot{Name: "rush", Rules: &planetv1.Rules{}})

	names, err := s.store.Names(s.T().Context())
	s.Require().NoError(err)
	s.Equal([]SetName{Default, "rush"}, names)
}

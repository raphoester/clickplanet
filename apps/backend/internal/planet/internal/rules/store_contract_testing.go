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
}

func (s *StoreContractSuite) save(kept Snapshot) {
	set, err := kept.Restore(s.Parts)
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), set))
}

func (s *StoreContractSuite) kept(name SetName) Snapshot {
	set, err := s.store.Set(s.T().Context(), name)
	s.Require().NoError(err)
	return set.TakeSnapshot()
}

func (s *StoreContractSuite) TestASetIsReadAsItWasKept() {
	rush := Named("rush", func(r *planetv1.Rules) { r.Clicks.Tempo = 3 })
	s.save(rush)

	kept := s.kept("rush")
	s.Equal(SetName("rush"), kept.Name)
	s.True(proto.Equal(rush.Rules, kept.Rules), "%v", kept.Rules)
}

func (s *StoreContractSuite) TestASetWithNoTollIsReadWithNone() {
	free := Named("free", func(r *planetv1.Rules) { r.Toll.Steps = nil })
	s.save(free)

	s.Empty(s.kept("free").Rules.GetToll().GetSteps())
}

func (s *StoreContractSuite) TestASetNeverKeptIsUnknown() {
	_, err := s.store.Set(s.T().Context(), "frozen")
	s.Require().ErrorIs(err, ErrUnknownSet)
}

func (s *StoreContractSuite) TestASetKeptAgainReplacesItsValues() {
	s.save(Named("rush", AsIs))
	quicker := Named("rush", func(r *planetv1.Rules) { r.Clicks.Tempo, r.Bonuses.Shield.PerTile = 4, 20 })
	s.save(quicker)

	s.True(proto.Equal(quicker.Rules, s.kept("rush").Rules))
}

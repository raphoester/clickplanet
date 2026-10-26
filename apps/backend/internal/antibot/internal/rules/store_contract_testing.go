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
}

func (s *StoreContractSuite) save(kept Snapshot) {
	set, err := kept.Restore(s.Parts)
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), set))
}

func (s *StoreContractSuite) requireKept(want Snapshot) {
	set, err := s.store.Set(s.T().Context(), want.Name)
	s.Require().NoError(err)
	kept := set.TakeSnapshot()
	s.Equal(want.Name, kept.Name)
	s.True(proto.Equal(want.Rules, kept.Rules), "%v", kept.Rules)
}

func (s *StoreContractSuite) TestASetIsReadAsItWasKept() {
	rush := Named("rush", func(r *antibotv1.Rules) {
		r.Cohort.CertainMembers = 50
		r.Metronome.Clock.MinCoherence = proto.Float64(0.9)
		r.Bans.Enforce = false
	})
	s.save(rush)

	s.requireKept(rush)
}

func (s *StoreContractSuite) TestASetNeverKeptIsUnknown() {
	_, err := s.store.Set(s.T().Context(), "frozen")
	s.Require().ErrorIs(err, ErrUnknownSet)
}

func (s *StoreContractSuite) TestASetKeptAgainReplacesItsValues() {
	s.save(Named("rush", AsIs))
	calmer := Named("rush", func(r *antibotv1.Rules) { r.Cohort.CertainMembers = 20 })
	s.save(calmer)

	s.requireKept(calmer)
}

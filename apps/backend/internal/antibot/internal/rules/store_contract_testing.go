//go:build testing

package rules

import (
	"github.com/stretchr/testify/suite"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

func (s *StoreContractSuite) save(kept Snapshot) {
	set, err := kept.Restore()
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), set))
}

var contractRush = Named("rush", func(s *Snapshot) {
	s.Cohort.CertainMembers = 50
	coherence := 0.9
	s.Metronome.Clock.MinCoherence = &coherence
	s.Bans.Enforce = false
})

func (s *StoreContractSuite) TestASetIsReadAsItWasKept() {
	s.save(contractRush)

	set, err := s.store.Set(s.T().Context(), "rush")
	s.Require().NoError(err)
	s.Equal(contractRush, set.TakeSnapshot())
}

func (s *StoreContractSuite) TestASetNeverKeptIsUnknown() {
	_, err := s.store.Set(s.T().Context(), "frozen")
	s.Require().ErrorIs(err, ErrUnknownSet)
}

func (s *StoreContractSuite) TestASetKeptAgainReplacesItsValues() {
	s.save(contractRush)
	calmer := Named("rush", func(s *Snapshot) { s.Cohort.CertainMembers = 20 })
	s.save(calmer)

	set, err := s.store.Set(s.T().Context(), "rush")
	s.Require().NoError(err)
	s.Equal(calmer, set.TakeSnapshot())
}

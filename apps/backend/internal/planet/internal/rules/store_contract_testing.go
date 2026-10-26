//go:build testing

package rules

import (
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

func contractRush() Snapshot {
	rush := TestSnapshot("rush")
	rush.Tempo = 3
	rush.Bonuses.Boxes = bonuses.Pace{MinInterval: time.Minute, MaxInterval: 2 * time.Minute, MaxChargesPerHour: 30}
	rush.Bonuses.Quiz.Enabled = true
	rush.Bonuses.Welcome = bonuses.WelcomeRules{Within: 10 * time.Minute, BoxAfter: time.Minute}
	return rush
}

func (s *StoreContractSuite) save(kept Snapshot) {
	set, err := kept.Restore()
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), set))
}

func (s *StoreContractSuite) TestASetIsReadAsItWasKept() {
	s.save(contractRush())

	set, err := s.store.Set(s.T().Context(), "rush")
	s.Require().NoError(err)
	s.Equal(contractRush(), set.TakeSnapshot())
}

func (s *StoreContractSuite) TestASetWithNoTollIsReadWithNone() {
	free := contractRush()
	free.Toll = nil
	s.save(free)

	set, err := s.store.Set(s.T().Context(), "rush")
	s.Require().NoError(err)
	s.Empty(set.TakeSnapshot().Toll)
}

func (s *StoreContractSuite) TestASetNeverKeptIsUnknown() {
	_, err := s.store.Set(s.T().Context(), "frozen")
	s.Require().ErrorIs(err, ErrUnknownSet)
}

func (s *StoreContractSuite) TestASetKeptAgainReplacesItsValues() {
	s.save(contractRush())
	quicker := contractRush()
	quicker.Tempo = 4
	quicker.Bonuses.Shield.PerTile = 20
	s.save(quicker)

	set, err := s.store.Set(s.T().Context(), "rush")
	s.Require().NoError(err)
	s.Equal(quicker, set.TakeSnapshot())
}

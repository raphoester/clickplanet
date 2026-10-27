//go:build testing

package calendar

import (
	"time"

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

var (
	contractStart   = time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)
	contractEnd     = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	contractDecided = time.Date(2026, 10, 28, 12, 0, 0, 123456000, time.UTC)
)

func (s *StoreContractSuite) save(decisions ...Decision) {
	for _, decision := range decisions {
		s.Require().NoError(s.store.Save(s.T().Context(), decision))
	}
}

func utc(decisions []Decision) []Decision {
	for i, d := range decisions {
		decisions[i] = Decision{
			Window:    Window{Number: d.Window.Number, StartsAt: d.Window.StartsAt.UTC(), EndsAt: d.Window.EndsAt.UTC()},
			DecidedAt: d.DecidedAt.UTC(),
		}
	}
	return decisions
}

func (s *StoreContractSuite) TestADecisionIsReadAsItWasKept() {
	decision := Decision{Window: Window{Number: 1, StartsAt: contractStart, EndsAt: contractEnd}, DecidedAt: contractDecided}
	s.save(decision)

	kept, found, err := s.store.Decision(s.T().Context(), 1)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Equal(utc([]Decision{decision}), utc([]Decision{kept}))
}

func (s *StoreContractSuite) TestASeasonNeverToldHasNoDecision() {
	_, found, err := s.store.Decision(s.T().Context(), 7)
	s.Require().NoError(err)
	s.False(found)
}

func (s *StoreContractSuite) TestADecisionKeptAgainReplacesTheSeasonsOne() {
	s.save(Decision{Window: Window{Number: 0, StartsAt: contractStart, EndsAt: contractEnd}, DecidedAt: contractDecided})
	moved := Decision{Window: Window{Number: 0, StartsAt: contractStart, EndsAt: contractEnd.AddDate(0, 0, 7)}, DecidedAt: contractDecided.Add(time.Hour)}
	s.save(moved)

	decisions, err := s.store.Decisions(s.T().Context())
	s.Require().NoError(err)
	s.Equal(utc([]Decision{moved}), utc(decisions))
}

func (s *StoreContractSuite) TestTheDecisionsAreEverySeasonsByNumber() {
	zero := Decision{Window: Window{Number: 0, StartsAt: contractStart, EndsAt: contractEnd}, DecidedAt: contractDecided}
	one := Decision{Window: Window{Number: 1, StartsAt: contractEnd, EndsAt: contractEnd.AddDate(0, 1, 0)}, DecidedAt: contractDecided}
	s.save(one, zero)

	decisions, err := s.store.Decisions(s.T().Context())
	s.Require().NoError(err)
	s.Equal(utc([]Decision{zero, one}), utc(decisions))
}

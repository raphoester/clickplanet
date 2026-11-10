//go:build testing

package departures

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

var contractStart = time.Date(2026, 11, 2, 12, 0, 0, 123_000_000, time.UTC)

func (s *StoreContractSuite) decide(account byte, decision Decision) {
	s.Require().NoError(s.store.Change(s.T().Context(), AccountID{15: account}, func(d Departure) Departure {
		return d.Decided(decision)
	}))
}

func (s *StoreContractSuite) departure(account byte) Departure {
	var kept Departure
	s.Require().NoError(s.store.Change(s.T().Context(), AccountID{15: account}, func(d Departure) Departure {
		kept = d
		return d
	}))
	return kept
}

func (s *StoreContractSuite) TestAnAccountNobodyAskedToDeleteHasTheZeroDeparture() {
	s.Equal(Departure{}, s.departure(1))
}

func (s *StoreContractSuite) TestADecisionIsKeptForItsAccountOnly() {
	s.decide(1, Leaves(contractStart))

	s.Equal(Departure{leaving: true, decidedAt: contractStart}, s.departure(1))
	s.Equal(Departure{}, s.departure(2))
}

func (s *StoreContractSuite) TestALaterDecisionReplacesTheKeptOneAndAnOlderOneDoesNot() {
	s.decide(1, Returns(contractStart.Add(time.Hour)))
	s.decide(1, Leaves(contractStart))

	s.Equal(Departure{decidedAt: contractStart.Add(time.Hour)}, s.departure(1), "the restore came after the request")

	s.decide(1, Leaves(contractStart.Add(2*time.Hour)))

	s.Equal(Departure{leaving: true, decidedAt: contractStart.Add(2 * time.Hour)}, s.departure(1))
}

func (s *StoreContractSuite) TestAForgottenAccountHasTheZeroDepartureAgain() {
	s.decide(1, Leaves(contractStart))
	s.decide(2, Leaves(contractStart))

	s.Require().NoError(s.store.Forget(s.T().Context(), AccountID{15: 1}))
	s.Require().NoError(s.store.Forget(s.T().Context(), AccountID{15: 1}), "twice is not an error")

	s.Equal(Departure{}, s.departure(1))
	s.True(s.departure(2).leaving)
}

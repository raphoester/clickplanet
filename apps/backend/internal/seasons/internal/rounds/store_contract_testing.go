//go:build testing

package rounds

import (
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
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
	contractDay     = Round{EndsAt: time.Date(2026, 10, 16, 21, 0, 0, 0, time.UTC)}
	contractNextDay = Round{EndsAt: contractDay.EndsAt.Add(Length)}
	contractFinale  = Round{EndsAt: time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC), Finale: true}
)

func (s *StoreContractSuite) open(rounds ...Round) {
	for _, round := range rounds {
		s.Require().NoError(s.store.Open(s.T().Context(), round))
	}
}

func (s *StoreContractSuite) unclosed(endedBy time.Time) []Round {
	ended, err := s.store.Unclosed(s.T().Context(), endedBy)
	s.Require().NoError(err)
	return ended
}

func (s *StoreContractSuite) TestARoundIsUnclosedFromItsEndUntilItIsClosed() {
	s.open(contractDay, contractNextDay, contractFinale)

	s.Empty(s.unclosed(contractDay.EndsAt.Add(-time.Second)))
	s.Equal([]Round{contractDay}, s.unclosed(contractDay.EndsAt))
	s.Equal([]Round{contractDay, contractNextDay}, s.unclosed(contractNextDay.EndsAt))
	s.Equal([]Round{contractDay, contractNextDay, contractFinale}, s.unclosed(contractFinale.EndsAt))
}

func (s *StoreContractSuite) TestARoundNeverOpenedIsNeverUnclosed() {
	s.Empty(s.unclosed(contractFinale.EndsAt.Add(Length)))
}

func (s *StoreContractSuite) TestARoundOpenedTwiceIsOneRound() {
	s.open(contractDay, contractDay)

	s.Equal([]Round{contractDay}, s.unclosed(contractDay.EndsAt))
	s.Equal(uint32(2), s.number(contractNextDay))
}

func (s *StoreContractSuite) TestAClosedRoundKeepsItsResultsAndIsNoLongerUnclosed() {
	s.open(contractDay)
	results := []Result{{Country: "fr", Rank: 1, Points: 25, Share: 0.3}, {Country: "de", Rank: 2, Points: 18, Share: 0.125}}

	closed, err := s.store.Close(s.T().Context(), contractDay, results)

	s.Require().NoError(err)
	s.True(closed)
	s.Equal([]Closed{{Round: contractDay, Number: 1, Results: results}}, s.closedIn(0))
	s.Empty(s.unclosed(contractNextDay.EndsAt))
}

func (s *StoreContractSuite) TestASecondCloseChangesNothing() {
	s.open(contractDay)
	first := []Result{{Country: "fr", Rank: 1, Points: 25, Share: 0.3}, {Country: "de", Rank: 2, Points: 18, Share: 0.1}}
	s.close(contractDay, first...)

	closed, err := s.store.Close(s.T().Context(), contractDay, []Result{{Country: "de", Rank: 1, Points: 25, Share: 0.5}})

	s.Require().NoError(err)
	s.False(closed, "only the first close closes the round")
	s.Equal([]Closed{{Round: contractDay, Number: 1, Results: first}}, s.closedIn(0))
}

func (s *StoreContractSuite) TestARoundNeverOpenedIsNotClosed() {
	closed, err := s.store.Close(s.T().Context(), contractDay, []Result{{Country: "fr", Rank: 1, Points: 25, Share: 0.3}})

	s.Require().NoError(err)
	s.False(closed)
	s.Empty(s.closedIn(0))
}

func (s *StoreContractSuite) TestOpeningAClosedRoundLeavesItClosed() {
	s.open(contractDay)
	results := []Result{{Country: "fr", Rank: 1, Points: 25, Share: 0.3}}
	s.close(contractDay, results...)

	s.open(contractDay)

	s.Empty(s.unclosed(contractDay.EndsAt))
	s.Equal([]Closed{{Round: contractDay, Number: 1, Results: results}}, s.closedIn(0))
}

func (s *StoreContractSuite) closedIn(season calendar.Number) []Closed {
	closed, err := s.store.ClosedIn(s.T().Context(), season)
	s.Require().NoError(err)
	return closed
}

func (s *StoreContractSuite) close(round Round, results ...Result) {
	_, err := s.store.Close(s.T().Context(), round, results)
	s.Require().NoError(err)
}

func (s *StoreContractSuite) TestASeasonListsTheRoundsItClosedInTheirOrder() {
	nextSeason := Round{Season: 1, EndsAt: contractFinale.EndsAt.Add(Length)}
	s.open(contractDay, contractNextDay, contractFinale, nextSeason)
	day := []Result{{Country: "fr", Rank: 1, Points: 25, Share: 0.3}, {Country: "de", Rank: 2, Points: 18, Share: 0.1}}
	nextDay := []Result{{Country: "de", Rank: 1, Points: 25, Share: 0.2}}
	s.close(contractNextDay, nextDay...)
	s.close(contractDay, day...)
	s.close(nextSeason, day...)

	s.Equal([]Closed{
		{Round: contractDay, Number: 1, Results: day},
		{Round: contractNextDay, Number: 2, Results: nextDay},
	}, s.closedIn(0), "the finale is not closed yet, and the next season is its own")
}

func (s *StoreContractSuite) TestARoundClosedWithNoResultsIsListedWithNone() {
	s.open(contractDay)
	s.close(contractDay)

	s.Equal([]Closed{{Round: contractDay, Number: 1}}, s.closedIn(0))
}

func (s *StoreContractSuite) TestASeasonWithNoRoundClosedListsNone() {
	s.open(contractDay)

	s.Empty(s.closedIn(0))
}

func (s *StoreContractSuite) number(round Round) uint32 {
	number, err := s.store.Number(s.T().Context(), round)
	s.Require().NoError(err)
	return number
}

func (s *StoreContractSuite) TestARoundIsNumberedAfterTheRoundsOfItsSeasonOpenedBeforeIt() {
	nextSeason := Round{Season: 1, EndsAt: contractNextDay.EndsAt.Add(Length)}
	s.open(contractDay, contractNextDay, nextSeason)

	s.Equal(uint32(1), s.number(contractDay))
	s.Equal(uint32(2), s.number(contractNextDay))
	s.Equal(uint32(3), s.number(contractFinale))
	s.Equal(uint32(1), s.number(nextSeason))
}

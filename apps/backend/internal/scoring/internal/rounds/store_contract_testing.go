//go:build testing

package rounds

import (
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar"
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
	contractDay     = Round{EndsAt: time.Date(2026, 10, 16, 21, 0, 0, 0, time.UTC), Times: 1}
	contractNextDay = Round{EndsAt: contractDay.EndsAt.Add(Length), Times: 1}
	contractFinale  = Round{
		EndsAt: time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC), Finale: true,
		Own: true, Entry: EntryID(uuid.MustParse("0192f5a0-7c00-7000-8000-000000000031")),
		StartsAt: time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC), Times: 3,
	}
	// contractBattle is a round of its own that ends with a day.
	contractBattle = Round{
		EndsAt: contractDay.EndsAt, Own: true, Entry: EntryID(uuid.MustParse("0192f5a0-7c00-7000-8000-000000000016")),
		StartsAt: contractDay.EndsAt.Add(-2 * time.Hour), Times: 2,
	}
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

func (s *StoreContractSuite) TestARoundOfItsOwnOpenedAgainByAnotherEntryKeepsTheFirstEntry() {
	other := contractFinale
	other.Entry = contractBattle.Entry
	s.open(contractFinale, other)

	s.Equal([]Round{contractFinale}, s.unclosed(contractFinale.EndsAt))
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
	nextSeason := Round{Season: 1, EndsAt: contractFinale.EndsAt.Add(Length), Times: 1}
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

func (s *StoreContractSuite) TestARoundIsNumberedAfterTheRoundsOfItsSeasonAndItsKindOpenedBeforeIt() {
	nextSeason := Round{Season: 1, EndsAt: contractNextDay.EndsAt.Add(Length), Times: 1}
	s.open(contractDay, contractBattle, contractNextDay, nextSeason)

	s.Equal(uint32(1), s.number(contractDay))
	s.Equal(uint32(2), s.number(contractNextDay))
	s.Equal(uint32(1), s.number(contractBattle))
	s.Equal(uint32(2), s.number(contractFinale))
	s.Equal(uint32(1), s.number(nextSeason))
}

func (s *StoreContractSuite) TestADayAndARoundOfItsOwnThatEndTogetherAreTwoRoundsTheDayFirst() {
	s.open(contractBattle, contractDay)

	s.Equal([]Round{contractDay, contractBattle}, s.unclosed(contractDay.EndsAt))

	s.close(contractBattle, Result{Country: "de", Rank: 1, Points: 50, Share: 0.4})
	s.close(contractDay, Result{Country: "fr", Rank: 1, Points: 25, Share: 0.3})
	s.Equal([]Closed{
		{Round: contractDay, Number: 1, Results: []Result{{Country: "fr", Rank: 1, Points: 25, Share: 0.3}}},
		{Round: contractBattle, Number: 1, Results: []Result{{Country: "de", Rank: 1, Points: 50, Share: 0.4}}},
	}, s.closedIn(0))
}

func (s *StoreContractSuite) firstEnd() (time.Time, bool) {
	first, found, err := s.store.FirstEnd(s.T().Context())
	s.Require().NoError(err)
	return first, found
}

func (s *StoreContractSuite) TestTheFirstEndIsTheEarliestOfTheRoundsLeftToClose() {
	_, found := s.firstEnd()
	s.False(found)

	s.open(contractNextDay, contractFinale, contractDay)
	first, found := s.firstEnd()
	s.True(found)
	s.True(contractDay.EndsAt.Equal(first), "%s", first)

	s.close(contractDay)
	first, _ = s.firstEnd()
	s.True(contractNextDay.EndsAt.Equal(first), "%s", first)
}

func (s *StoreContractSuite) TestASeasonEndsOnce() {
	season := calendar.Season{Number: 0, LastDayEndsAt: contractDay.EndsAt, EndsAt: contractFinale.EndsAt}

	ended, err := s.store.Ended(s.T().Context(), 0)
	s.Require().NoError(err)
	s.False(ended)

	first, err := s.store.End(s.T().Context(), season)
	s.Require().NoError(err)
	again, err := s.store.End(s.T().Context(), season)
	s.Require().NoError(err)

	s.True(first)
	s.False(again, "only the first end ends the season")
	ended, err = s.store.Ended(s.T().Context(), 0)
	s.Require().NoError(err)
	s.True(ended)
	ended, err = s.store.Ended(s.T().Context(), 1)
	s.Require().NoError(err)
	s.False(ended)
}

// TakesContractSuite pins what a round of its own counts of its takes. CountedIn reads it back, since the write side
// reads nothing.
type TakesContractSuite struct {
	suite.Suite

	NewStores func() (Store, Takes)
	CountedIn func(round Round) Counted

	store Store
	takes Takes
}

// Counted is the tiles a round counted for each flag, and for each account and flag.
type Counted struct {
	Countries map[Country]uint64
	Players   map[AccountID]map[Country]uint64
}

func (s *TakesContractSuite) SetupTest() {
	s.store, s.takes = s.NewStores()
}

var (
	contractPlayer = AccountID{1}
	contractOther  = AccountID{2}
)

func (s *TakesContractSuite) count(takes ...Take) {
	for _, take := range takes {
		s.Require().NoError(s.takes.Count(s.T().Context(), take))
	}
}

func (s *TakesContractSuite) TestARoundOfItsOwnCountsTheTakesOfItsWindowOnce() {
	s.Require().NoError(s.store.Open(s.T().Context(), contractFinale))
	s.Require().NoError(s.store.Open(s.T().Context(), contractDay))
	at := contractFinale.StartsAt

	s.count(
		Take{Account: contractPlayer, Country: "fr", At: at.Add(-time.Second), Position: 1},
		Take{Account: contractPlayer, Country: "fr", At: at, Position: 2},
		Take{Account: contractPlayer, Country: "fr", At: at.Add(time.Minute), Position: 3},
		Take{Account: contractPlayer, Country: "fr", At: at.Add(time.Minute), Position: 3},
		Take{Account: contractPlayer, Country: "be", At: at.Add(time.Hour), Position: 4},
		Take{Account: contractOther, Country: "de", At: at.Add(time.Hour), Position: 5},
		Take{Account: contractOther, Country: "de", At: contractFinale.EndsAt, Position: 6},
	)

	s.Equal(Counted{
		Countries: map[Country]uint64{"fr": 2, "be": 1, "de": 1},
		Players: map[AccountID]map[Country]uint64{
			contractPlayer: {"fr": 2, "be": 1},
			contractOther:  {"de": 1},
		},
	}, s.CountedIn(contractFinale), "from its start, to before its end, a take told again once")
	s.Empty(s.CountedIn(contractDay).Countries, "a day counts no takes")
}

func (s *TakesContractSuite) TestAnAccountDeletedIsForgottenAndItsFlagKeepsItsTiles() {
	s.Require().NoError(s.store.Open(s.T().Context(), contractFinale))
	s.count(
		Take{Account: contractPlayer, Country: "fr", At: contractFinale.StartsAt, Position: 1},
		Take{Account: contractOther, Country: "fr", At: contractFinale.StartsAt, Position: 2},
	)

	s.Require().NoError(s.takes.ForgetAccount(s.T().Context(), contractPlayer))

	s.Equal(Counted{
		Countries: map[Country]uint64{"fr": 2},
		Players:   map[AccountID]map[Country]uint64{contractOther: {"fr": 1}},
	}, s.CountedIn(contractFinale))
}

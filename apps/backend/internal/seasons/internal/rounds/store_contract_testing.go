//go:build testing

package rounds

import (
	"sync"
	"time"

	"github.com/stretchr/testify/suite"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore  func() Store
	ResultsOf func(store Store, round Round) []Result

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

func (s *StoreContractSuite) snapshot(round Round, held map[Country]uint32) {
	s.Require().NoError(s.store.RecordSnapshot(s.T().Context(), round, Snapshot{Tiles: 100, Held: held}))
}

func (s *StoreContractSuite) held(round Round) map[Country]uint64 {
	held, err := s.store.Held(s.T().Context(), round)
	s.Require().NoError(err)
	return held
}

func (s *StoreContractSuite) unclosed(endedBy time.Time) []Round {
	ended, err := s.store.Unclosed(s.T().Context(), endedBy)
	s.Require().NoError(err)
	return ended
}

func (s *StoreContractSuite) TestARoundNobodyCountedHoldsNothing() {
	s.Empty(s.held(contractDay))
}

func (s *StoreContractSuite) TestEachSnapshotAddsWhatEachCountryHeld() {
	s.snapshot(contractDay, map[Country]uint32{"fr": 3, "de": 1})
	s.snapshot(contractDay, map[Country]uint32{"fr": 2})

	s.Equal(map[Country]uint64{"fr": 5, "de": 1}, s.held(contractDay))
}

func (s *StoreContractSuite) TestACountryThatHeldNothingIsNotKept() {
	s.snapshot(contractDay, map[Country]uint32{"fr": 2, "de": 0})

	s.Equal(map[Country]uint64{"fr": 2}, s.held(contractDay))
}

func (s *StoreContractSuite) TestEachRoundKeepsItsOwnSnapshots() {
	nextSeason := Round{Season: 1, EndsAt: contractDay.EndsAt}
	s.snapshot(contractDay, map[Country]uint32{"fr": 1})
	s.snapshot(contractNextDay, map[Country]uint32{"de": 2})
	s.snapshot(nextSeason, map[Country]uint32{"it": 3})

	s.Equal(map[Country]uint64{"fr": 1}, s.held(contractDay))
	s.Equal(map[Country]uint64{"de": 2}, s.held(contractNextDay))
	s.Equal(map[Country]uint64{"it": 3}, s.held(nextSeason))
}

func (s *StoreContractSuite) TestARoundIsUnclosedFromItsEndUntilItIsClosed() {
	s.snapshot(contractDay, map[Country]uint32{"fr": 1})
	s.snapshot(contractNextDay, map[Country]uint32{"fr": 1})
	s.snapshot(contractFinale, map[Country]uint32{"fr": 1})

	s.Empty(s.unclosed(contractDay.EndsAt.Add(-time.Second)))
	s.Equal([]Round{contractDay}, s.unclosed(contractDay.EndsAt))
	s.Equal([]Round{contractDay, contractNextDay}, s.unclosed(contractNextDay.EndsAt))
	s.Equal([]Round{contractDay, contractNextDay, contractFinale}, s.unclosed(contractFinale.EndsAt))
}

func (s *StoreContractSuite) TestARoundNobodyCountedIsNeverUnclosed() {
	s.Empty(s.unclosed(contractFinale.EndsAt.Add(Length)))
}

func (s *StoreContractSuite) TestAClosedRoundKeepsItsResultsAndIsNoLongerUnclosed() {
	s.snapshot(contractDay, map[Country]uint32{"fr": 2, "de": 1})
	results := []Result{{Country: "fr", Rank: 1, Points: 25}, {Country: "de", Rank: 2, Points: 18}}

	s.Require().NoError(s.store.Close(s.T().Context(), contractDay, results))

	s.Equal(results, s.ResultsOf(s.store, contractDay))
	s.Empty(s.unclosed(contractNextDay.EndsAt))
}

func (s *StoreContractSuite) TestASecondCloseChangesNothing() {
	s.snapshot(contractDay, map[Country]uint32{"fr": 2, "de": 1})
	first := []Result{{Country: "fr", Rank: 1, Points: 25}, {Country: "de", Rank: 2, Points: 18}}
	s.Require().NoError(s.store.Close(s.T().Context(), contractDay, first))

	s.Require().NoError(s.store.Close(s.T().Context(), contractDay, []Result{{Country: "de", Rank: 1, Points: 25}}))

	s.Equal(first, s.ResultsOf(s.store, contractDay))
}

func (s *StoreContractSuite) number(round Round) uint32 {
	number, err := s.store.Number(s.T().Context(), round)
	s.Require().NoError(err)
	return number
}

func (s *StoreContractSuite) TestARoundIsNumberedAfterTheRoundsOfItsSeasonCountedBeforeIt() {
	nextSeason := Round{Season: 1, EndsAt: contractNextDay.EndsAt.Add(Length)}
	s.snapshot(contractDay, map[Country]uint32{"fr": 1})
	s.snapshot(contractNextDay, map[Country]uint32{"fr": 1})
	s.snapshot(nextSeason, map[Country]uint32{"fr": 1})

	s.Equal(uint32(1), s.number(contractDay))
	s.Equal(uint32(2), s.number(contractNextDay))
	s.Equal(uint32(3), s.number(contractFinale))
	s.Equal(uint32(1), s.number(nextSeason))
}

func (s *StoreContractSuite) TestSnapshotsAtOnceAreAllCounted() {
	var wg sync.WaitGroup
	for range 40 {
		wg.Go(func() {
			s.NoError(s.store.RecordSnapshot(s.T().Context(), contractDay,
				Snapshot{Tiles: 100, Held: map[Country]uint32{"fr": 1, "de": 2}}))
		})
	}
	wg.Wait()

	s.Equal(map[Country]uint64{"fr": 40, "de": 80}, s.held(contractDay))
}

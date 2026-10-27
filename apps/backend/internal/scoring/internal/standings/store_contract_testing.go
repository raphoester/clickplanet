//go:build testing

package standings

import (
	"sync"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store
	TallyOf  func(store Store, season calendar.Number, account AccountID) Tally

	store Store
	taken uint64
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
	s.taken = 0
}

var contractAt = time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)

func contractAccount(n byte) AccountID {
	return AccountID{0: 0x01, 15: n}
}

func (s *StoreContractSuite) take(season calendar.Number, account byte, country Country, tiles int) {
	for range tiles {
		s.Require().NoError(s.store.RecordTake(s.T().Context(), season,
			Take{Account: contractAccount(account), Country: country, At: contractAt, Position: s.taken}))
		s.taken++
	}
}

func (s *StoreContractSuite) tally(season calendar.Number, account byte) Tally {
	return s.TallyOf(s.store, season, contractAccount(account))
}

func (s *StoreContractSuite) TestAnAccountThatTookNothingHasAnEmptyTally() {
	s.True(s.tally(0, 1).Empty())
}

func (s *StoreContractSuite) TestEachTakeAddsOneTileToItsFlag() {
	s.take(0, 1, "fr", 3)
	s.take(0, 1, "de", 2)

	s.Equal(Tally{Main: "fr", Tiles: map[Country]uint64{"fr": 3, "de": 2}, TakesBelow: 5}, s.tally(0, 1))
}

func (s *StoreContractSuite) TestAFlagThatOvertakesTheMainOneBecomesIt() {
	s.take(0, 1, "fr", 2)
	s.take(0, 1, "de", 2)
	s.Equal(Country("fr"), s.tally(0, 1).Main, "a tie keeps the flag that got there first")

	s.take(0, 1, "de", 1)
	s.Equal(Country("de"), s.tally(0, 1).Main)
}

func (s *StoreContractSuite) TestEachSeasonCountsItsOwnTakes() {
	s.take(0, 1, "fr", 3)
	s.take(1, 1, "de", 1)

	s.Equal(Tally{Main: "fr", Tiles: map[Country]uint64{"fr": 3}, TakesBelow: 3}, s.tally(0, 1))
	s.Equal(Tally{Main: "de", Tiles: map[Country]uint64{"de": 1}, TakesBelow: 4}, s.tally(1, 1))
}

func (s *StoreContractSuite) TestADeletedAccountLeavesEverySeasonAndTheOthersStay() {
	s.take(0, 1, "fr", 2)
	s.take(1, 1, "fr", 1)
	s.take(0, 2, "de", 1)

	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), contractAccount(1)))

	s.True(s.tally(0, 1).Empty())
	s.True(s.tally(1, 1).Empty())
	s.Equal(Tally{Main: "de", Tiles: map[Country]uint64{"de": 1}, TakesBelow: 4}, s.tally(0, 2))
}

func (s *StoreContractSuite) TestDeletingAnAccountThatTookNothingIsNotAnError() {
	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), contractAccount(9)))
}

func (s *StoreContractSuite) TestATakeToldAgainAfterItWasKeptIsNotCountedAgain() {
	for _, position := range []uint64{7, 7, 3} {
		s.Require().NoError(s.store.RecordTake(s.T().Context(), 0,
			Take{Account: contractAccount(1), Country: "fr", At: contractAt, Position: position}))
	}

	s.Equal(Tally{Main: "fr", Tiles: map[Country]uint64{"fr": 1}, TakesBelow: 8}, s.tally(0, 1))
}

func (s *StoreContractSuite) TestTheTakesOfManyAccountsAtOnceAreAllCounted() {
	var wg sync.WaitGroup
	for account := range byte(20) {
		wg.Go(func() {
			for position, country := range []Country{"fr", "de", "de"} {
				s.NoError(s.store.RecordTake(s.T().Context(), 0,
					Take{Account: contractAccount(account), Country: country, At: contractAt, Position: uint64(position)}))
			}
		})
	}
	wg.Wait()

	for account := range byte(20) {
		s.Equal(Tally{Main: "de", Tiles: map[Country]uint64{"fr": 1, "de": 2}, TakesBelow: 3}, s.tally(0, account))
	}
}

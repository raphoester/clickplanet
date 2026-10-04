//go:build testing

package standings

import (
	"sync"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store
	TallyOf  func(store Store, season calendar.Number, account AccountID) Tally

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

var contractAt = time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)

func contractAccount(n byte) AccountID {
	return AccountID{0: 0x01, 15: n}
}

func (s *StoreContractSuite) take(season calendar.Number, account byte, country Country, tiles int) {
	for range tiles {
		s.Require().NoError(s.store.RecordTake(s.T().Context(), season,
			Take{Account: contractAccount(account), Country: country, At: contractAt}))
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

	s.Equal(Tally{Main: "fr", Tiles: map[Country]uint64{"fr": 3, "de": 2}}, s.tally(0, 1))
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

	s.Equal(Tally{Main: "fr", Tiles: map[Country]uint64{"fr": 3}}, s.tally(0, 1))
	s.Equal(Tally{Main: "de", Tiles: map[Country]uint64{"de": 1}}, s.tally(1, 1))
}

func (s *StoreContractSuite) TestADeletedAccountLeavesEverySeasonAndTheOthersStay() {
	s.take(0, 1, "fr", 2)
	s.take(1, 1, "fr", 1)
	s.take(0, 2, "de", 1)

	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), contractAccount(1)))

	s.True(s.tally(0, 1).Empty())
	s.True(s.tally(1, 1).Empty())
	s.Equal(Tally{Main: "de", Tiles: map[Country]uint64{"de": 1}}, s.tally(0, 2))
}

func (s *StoreContractSuite) TestDeletingAnAccountThatTookNothingIsNotAnError() {
	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), contractAccount(9)))
}

func (s *StoreContractSuite) TestTakesAtOnceAreAllCounted() {
	var wg sync.WaitGroup
	for i := range 40 {
		country := Country("fr")
		if i%2 == 1 {
			country = "de"
		}
		wg.Go(func() {
			s.NoError(s.store.RecordTake(s.T().Context(), 0, Take{Account: contractAccount(1), Country: country, At: contractAt}))
		})
	}
	wg.Wait()

	s.Equal(map[Country]uint64{"fr": 20, "de": 20}, s.tally(0, 1).Tiles)
}

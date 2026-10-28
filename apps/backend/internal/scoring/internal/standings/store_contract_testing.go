//go:build testing

package standings

import (
	"sync"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store
	TallyOf  func(store Store, book books.BookID, account AccountID) Tally

	store Store
	taken uint64
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
	s.taken = 0
}

var (
	contractAt   = time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	contractBook = books.BookID("season-0")
	contractNext = books.BookID("season-1")
)

func contractAccount(n byte) AccountID {
	return AccountID{0: 0x01, 15: n}
}

func (s *StoreContractSuite) take(book books.BookID, account byte, country Country, tiles int) {
	for range tiles {
		s.Require().NoError(s.store.RecordTake(s.T().Context(), book,
			Take{Account: contractAccount(account), Country: country, At: contractAt, Position: s.taken}))
		s.taken++
	}
}

func (s *StoreContractSuite) tally(book books.BookID, account byte) Tally {
	return s.TallyOf(s.store, book, contractAccount(account))
}

func (s *StoreContractSuite) TestAnAccountThatTookNothingHasAnEmptyTally() {
	s.True(s.tally(contractBook, 1).Empty())
}

func (s *StoreContractSuite) TestEachTakeAddsOneTileToItsFlag() {
	s.take(contractBook, 1, "fr", 3)
	s.take(contractBook, 1, "de", 2)

	s.Equal(Tally{Main: "fr", Tiles: map[Country]uint64{"fr": 3, "de": 2}, TakesBelow: 5}, s.tally(contractBook, 1))
}

func (s *StoreContractSuite) TestAFlagThatOvertakesTheMainOneBecomesIt() {
	s.take(contractBook, 1, "fr", 2)
	s.take(contractBook, 1, "de", 2)
	s.Equal(Country("fr"), s.tally(contractBook, 1).Main, "a tie keeps the flag that got there first")

	s.take(contractBook, 1, "de", 1)
	s.Equal(Country("de"), s.tally(contractBook, 1).Main)
}

func (s *StoreContractSuite) TestEachBookCountsItsOwnTakes() {
	s.take(contractBook, 1, "fr", 3)
	s.take(contractNext, 1, "de", 1)

	s.Equal(Tally{Main: "fr", Tiles: map[Country]uint64{"fr": 3}, TakesBelow: 3}, s.tally(contractBook, 1))
	s.Equal(Tally{Main: "de", Tiles: map[Country]uint64{"de": 1}, TakesBelow: 4}, s.tally(contractNext, 1))
}

func (s *StoreContractSuite) TestADeletedAccountLeavesEveryBookAndTheOthersStay() {
	s.take(contractBook, 1, "fr", 2)
	s.take(contractNext, 1, "fr", 1)
	s.take(contractBook, 2, "de", 1)

	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), contractAccount(1)))

	s.True(s.tally(contractBook, 1).Empty())
	s.True(s.tally(contractNext, 1).Empty())
	s.Equal(Tally{Main: "de", Tiles: map[Country]uint64{"de": 1}, TakesBelow: 4}, s.tally(contractBook, 2))
}

func (s *StoreContractSuite) TestDeletingAnAccountThatTookNothingIsNotAnError() {
	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), contractAccount(9)))
}

func (s *StoreContractSuite) TestATakeToldAgainAfterItWasKeptIsNotCountedAgain() {
	for _, position := range []uint64{7, 7, 3} {
		s.Require().NoError(s.store.RecordTake(s.T().Context(), contractBook,
			Take{Account: contractAccount(1), Country: "fr", At: contractAt, Position: position}))
	}

	s.Equal(Tally{Main: "fr", Tiles: map[Country]uint64{"fr": 1}, TakesBelow: 8}, s.tally(contractBook, 1))
}

func (s *StoreContractSuite) TestTheTakesOfManyAccountsAtOnceAreAllCounted() {
	var wg sync.WaitGroup
	for account := range byte(20) {
		wg.Go(func() {
			for position, country := range []Country{"fr", "de", "de"} {
				s.NoError(s.store.RecordTake(s.T().Context(), contractBook,
					Take{Account: contractAccount(account), Country: country, At: contractAt, Position: uint64(position)}))
			}
		})
	}
	wg.Wait()

	for account := range byte(20) {
		s.Equal(Tally{Main: "de", Tiles: map[Country]uint64{"fr": 1, "de": 2}, TakesBelow: 3}, s.tally(contractBook, account))
	}
}

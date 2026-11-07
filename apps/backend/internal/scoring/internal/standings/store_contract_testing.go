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

// take records the tiles as one batch, each at a position of its own.
func (s *StoreContractSuite) take(book books.BookID, account byte, country Country, tiles int) {
	takes := make([]Take, 0, tiles)
	for range tiles {
		takes = append(takes, Take{Country: country, Position: s.taken})
		s.taken++
	}
	s.record(book, account, takes...)
}

func (s *StoreContractSuite) record(book books.BookID, account byte, takes ...Take) {
	told := make([]Take, 0, len(takes))
	for _, take := range takes {
		take.Account, take.At = contractAccount(account), contractAt
		told = append(told, take)
	}
	s.Require().NoError(s.store.RecordTakes(s.T().Context(), book, contractAccount(account), told))
}

func (s *StoreContractSuite) tally(book books.BookID, account byte) Tally {
	return s.TallyOf(s.store, book, contractAccount(account))
}

func (s *StoreContractSuite) TestAnAccountThatTookNothingHasAnEmptyTally() {
	s.True(s.tally(contractBook, 1).Empty())
}

func (s *StoreContractSuite) TestNoTakeRecordsNothing() {
	s.record(contractBook, 1)

	s.True(s.tally(contractBook, 1).Empty())
}

func (s *StoreContractSuite) TestEachTakeAddsOneTileToItsFlag() {
	s.take(contractBook, 1, "fr", 3)
	s.take(contractBook, 1, "de", 2)

	s.Equal(Tally{Main: "fr", Tiles: map[Country]uint64{"fr": 3, "de": 2}, TakesBelow: TakeMark{Position: 4, Index: 1}}, s.tally(contractBook, 1))
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

	s.Equal(Tally{Main: "fr", Tiles: map[Country]uint64{"fr": 3}, TakesBelow: TakeMark{Position: 2, Index: 1}}, s.tally(contractBook, 1))
	s.Equal(Tally{Main: "de", Tiles: map[Country]uint64{"de": 1}, TakesBelow: TakeMark{Position: 3, Index: 1}}, s.tally(contractNext, 1))
}

func (s *StoreContractSuite) TestADeletedAccountLeavesEveryBookAndTheOthersStay() {
	s.take(contractBook, 1, "fr", 2)
	s.take(contractNext, 1, "fr", 1)
	s.take(contractBook, 2, "de", 1)

	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), contractAccount(1)))

	s.True(s.tally(contractBook, 1).Empty())
	s.True(s.tally(contractNext, 1).Empty())
	s.Equal(Tally{Main: "de", Tiles: map[Country]uint64{"de": 1}, TakesBelow: TakeMark{Position: 3, Index: 1}}, s.tally(contractBook, 2))
}

func (s *StoreContractSuite) TestDeletingAnAccountThatTookNothingIsNotAnError() {
	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), contractAccount(9)))
}

func (s *StoreContractSuite) TestATakeToldAgainAfterItWasKeptIsNotCountedAgain() {
	for _, position := range []uint64{7, 7, 3} {
		s.record(contractBook, 1, Take{Country: "fr", Position: position})
	}

	s.Equal(Tally{Main: "fr", Tiles: map[Country]uint64{"fr": 1}, TakesBelow: TakeMark{Position: 7, Index: 1}}, s.tally(contractBook, 1))
}

func (s *StoreContractSuite) TestEachTakeOfASpreadIsCountedOnceThoughTheyShareAPosition() {
	takes := []Take{
		{Country: "de", Position: 8},
		{Country: "fr", Position: 9, Index: 0},
		{Country: "fr", Position: 9, Index: 1},
		{Country: "fr", Position: 9, Index: 2},
	}
	for range 2 {
		for _, take := range takes {
			s.record(contractBook, 1, take)
		}
	}

	s.Equal(Tally{
		Main: "fr", Tiles: map[Country]uint64{"de": 1, "fr": 3}, TakesBelow: TakeMark{Position: 9, Index: 3},
	}, s.tally(contractBook, 1))
}

func (s *StoreContractSuite) TestABatchToldTwiceIsCountedOnce() {
	batch := []Take{
		{Country: "fr", Position: 4},
		{Country: "fr", Position: 5, Index: 0},
		{Country: "fr", Position: 5, Index: 1},
		{Country: "de", Position: 6},
	}

	s.record(contractBook, 1, batch...)
	s.record(contractBook, 1, batch...)
	s.record(contractBook, 1, batch[2:]...)

	s.Equal(Tally{Main: "fr", Tiles: map[Country]uint64{"fr": 3, "de": 1}, TakesBelow: TakeMark{Position: 6, Index: 1}},
		s.tally(contractBook, 1), "a batch, then again, then a slice of it one event at a time")
}

func (s *StoreContractSuite) TestABatchWhoseFirstTakesWereCountedCountsOnlyTheRest() {
	s.record(contractBook, 1, Take{Country: "fr", Position: 4}, Take{Country: "fr", Position: 5, Index: 0})

	s.record(contractBook, 1,
		Take{Country: "fr", Position: 4},
		Take{Country: "fr", Position: 5, Index: 0},
		Take{Country: "fr", Position: 5, Index: 1},
		Take{Country: "de", Position: 7},
	)

	s.Equal(Tally{Main: "fr", Tiles: map[Country]uint64{"fr": 3, "de": 1}, TakesBelow: TakeMark{Position: 7, Index: 1}},
		s.tally(contractBook, 1))
}

func (s *StoreContractSuite) TestABatchOfSeveralFlagsMovesTheMainFlagOnceToTheOneWithTheMostTiles() {
	s.take(contractBook, 1, "fr", 2)

	s.record(contractBook, 1,
		Take{Country: "de", Position: 10},
		Take{Country: "it", Position: 11},
		Take{Country: "de", Position: 12},
		Take{Country: "it", Position: 13},
		Take{Country: "it", Position: 14},
	)

	s.Equal(Tally{Main: "it", Tiles: map[Country]uint64{"fr": 2, "de": 2, "it": 3}, TakesBelow: TakeMark{Position: 14, Index: 1}},
		s.tally(contractBook, 1), "de reached fr's two and kept behind it; it passed both")

	s.record(contractBook, 1, Take{Country: "fr", Position: 15}, Take{Country: "fr", Position: 16})
	s.Equal(Country("fr"), s.tally(contractBook, 1).Main, "the main flag moves back")
}

func (s *StoreContractSuite) TestTheTakesOfManyAccountsAtOnceAreAllCounted() {
	var wg sync.WaitGroup
	for account := range byte(20) {
		wg.Go(func() {
			for position, country := range []Country{"fr", "de", "de"} {
				s.NoError(s.store.RecordTakes(s.T().Context(), contractBook, contractAccount(account),
					[]Take{{Account: contractAccount(account), Country: country, At: contractAt, Position: uint64(position)}}))
			}
		})
	}
	wg.Wait()

	for account := range byte(20) {
		s.Equal(Tally{Main: "de", Tiles: map[Country]uint64{"fr": 1, "de": 2}, TakesBelow: TakeMark{Position: 2, Index: 1}},
			s.tally(contractBook, account))
	}
}

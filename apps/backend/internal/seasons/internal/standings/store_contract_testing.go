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

func (s *StoreContractSuite) line(season calendar.Number, account byte) Line {
	line, err := s.store.Line(s.T().Context(), season, contractAccount(account))
	s.Require().NoError(err)
	return line
}

func (s *StoreContractSuite) lines(season calendar.Number, country Country, from Cursor, limit int) []Line {
	lines, err := s.store.Lines(s.T().Context(), season, country, from, limit)
	s.Require().NoError(err)
	return lines
}

func (s *StoreContractSuite) TestAnAccountThatTookNothingHasNoLine() {
	_, err := s.store.Line(s.T().Context(), 0, contractAccount(1))

	s.Require().ErrorIs(err, ErrNoLine)
	s.Empty(s.lines(0, "", Start, 10))
}

func (s *StoreContractSuite) TestTheLineIsTheMainFlagAndItsTiles() {
	s.take(0, 1, "fr", 3)
	s.take(0, 1, "de", 2)

	s.Equal(Line{Account: contractAccount(1), Country: "fr", Tiles: 3}, s.line(0, 1))
}

func (s *StoreContractSuite) TestAFlagThatOvertakesTheMainOneBecomesIt() {
	s.take(0, 1, "fr", 2)
	s.take(0, 1, "de", 2)
	s.Equal(Country("fr"), s.line(0, 1).Country, "a tie keeps the flag that got there first")

	s.take(0, 1, "de", 1)
	s.Equal(Line{Account: contractAccount(1), Country: "de", Tiles: 3}, s.line(0, 1))
	s.Equal([]Line{{Account: contractAccount(1), Country: "de", Tiles: 3}}, s.lines(0, "", Start, 10))
	s.Empty(s.lines(0, "fr", Start, 10))
}

func (s *StoreContractSuite) TestEachSeasonCountsItsOwnTakes() {
	s.take(0, 1, "fr", 3)
	s.take(1, 1, "de", 1)

	s.Equal(Line{Account: contractAccount(1), Country: "fr", Tiles: 3}, s.line(0, 1))
	s.Equal(Line{Account: contractAccount(1), Country: "de", Tiles: 1}, s.line(1, 1))
	s.Len(s.lines(1, "", Start, 10), 1)
}

func (s *StoreContractSuite) TestLinesAreBestFirstThenByAccount() {
	s.take(0, 3, "fr", 2)
	s.take(0, 1, "de", 5)
	s.take(0, 2, "fr", 2)
	s.take(0, 4, "it", 1)

	s.Equal([]Line{
		{Account: contractAccount(1), Country: "de", Tiles: 5},
		{Account: contractAccount(2), Country: "fr", Tiles: 2},
		{Account: contractAccount(3), Country: "fr", Tiles: 2},
		{Account: contractAccount(4), Country: "it", Tiles: 1},
	}, s.lines(0, "", Start, 10))
}

func (s *StoreContractSuite) TestACountryKeepsTheAccountsWhoseMainFlagItIs() {
	s.take(0, 1, "de", 5)
	s.take(0, 1, "fr", 4)
	s.take(0, 2, "fr", 2)

	s.Equal([]Line{{Account: contractAccount(2), Country: "fr", Tiles: 2}}, s.lines(0, "fr", Start, 10))
}

func (s *StoreContractSuite) TestLinesArePagedFromACursor() {
	s.take(0, 1, "de", 3)
	s.take(0, 2, "fr", 2)
	s.take(0, 3, "fr", 2)
	s.take(0, 4, "it", 1)

	first := s.lines(0, "", Start, 2)
	s.Equal([]AccountID{contractAccount(1), contractAccount(2)}, accountsOf(first))
	rest := s.lines(0, "", first[1].Cursor(), 2)
	s.Equal([]AccountID{contractAccount(3), contractAccount(4)}, accountsOf(rest))
	s.Empty(s.lines(0, "", rest[1].Cursor(), 2))
}

func (s *StoreContractSuite) TestADeletedAccountLeavesEverySeasonAndTheOthersStay() {
	s.take(0, 1, "fr", 2)
	s.take(1, 1, "fr", 1)
	s.take(0, 2, "de", 1)

	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), contractAccount(1)))

	_, err := s.store.Line(s.T().Context(), 1, contractAccount(1))
	s.Require().ErrorIs(err, ErrNoLine)
	s.Equal([]AccountID{contractAccount(2)}, accountsOf(s.lines(0, "", Start, 10)))
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

	s.Equal(uint64(20), s.line(0, 1).Tiles)
	s.Len(s.lines(0, "", Start, 10), 1)
}

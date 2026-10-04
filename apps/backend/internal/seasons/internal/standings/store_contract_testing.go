//go:build testing

package standings

import (
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store
	TallyOf  func(store Store, season calendar.Number, account AccountID) Tally

	store Store
	begun bool
	next  Position
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
	s.begun, s.next = false, 0
}

func (s *StoreContractSuite) begin() {
	if !s.begun {
		s.Require().NoError(s.store.Begin(s.T().Context(), 0))
		s.begun = true
	}
}

var contractAt = time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)

var contractSeasons = calendar.New(calendar.Config{List: []calendar.Entry{
	{Number: 0, EndsAt: contractAt.Add(24 * time.Hour), Finale: time.Hour},
	{Number: 1, EndsAt: contractAt.Add(30 * 24 * time.Hour), Finale: time.Hour},
}})

func contractAccount(n byte) AccountID {
	return AccountID{0: 0x01, 15: n}
}

func contractTime(season calendar.Number) time.Time {
	return contractAt.Add(time.Duration(season) * 48 * time.Hour)
}

func (s *StoreContractSuite) batch(entries ...Entry) Batch {
	from, next := s.next, s.next
	if len(entries) > 0 {
		from, next = entries[0].position, entries[len(entries)-1].position+1
	}
	batch, err := BatchOf(from, next, entries)
	s.Require().NoError(err)
	return batch
}

func (s *StoreContractSuite) count(batch Batch) {
	s.begin()
	s.Require().NoError(s.store.Count(s.T().Context(), batch, contractSeasons))
	s.next = batch.Next()
}

func (s *StoreContractSuite) entry(season calendar.Number, account byte, country Country, reverted bool) Entry {
	entry := EntryOf(s.next, Take{Account: contractAccount(account), Country: country, At: contractTime(season)}, reverted)
	s.next++
	return entry
}

func (s *StoreContractSuite) take(season calendar.Number, account byte, country Country, tiles int) {
	entries := make([]Entry, 0, tiles)
	for range tiles {
		entries = append(entries, s.entry(season, account, country, false))
	}
	s.count(s.batch(entries...))
}

func (s *StoreContractSuite) tally(season calendar.Number, account byte) Tally {
	return s.TallyOf(s.store, season, contractAccount(account))
}

func (s *StoreContractSuite) position() Position {
	position, err := s.store.Position(s.T().Context())
	s.Require().NoError(err)
	return position
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

func (s *StoreContractSuite) TestAFlagOvertakesTheMainOneInsideOneBatch() {
	s.count(s.batch(
		s.entry(0, 1, "fr", false),
		s.entry(0, 1, "de", false),
		s.entry(0, 1, "de", false),
		s.entry(0, 2, "it", false),
	))

	s.Equal(Tally{Main: "de", Tiles: map[Country]uint64{"fr": 1, "de": 2}}, s.tally(0, 1))
	s.Equal(Tally{Main: "it", Tiles: map[Country]uint64{"it": 1}}, s.tally(0, 2))
}

func (s *StoreContractSuite) TestEachSeasonCountsItsOwnTakes() {
	s.take(0, 1, "fr", 3)
	s.take(1, 1, "de", 1)

	s.Equal(Tally{Main: "fr", Tiles: map[Country]uint64{"fr": 3}}, s.tally(0, 1))
	s.Equal(Tally{Main: "de", Tiles: map[Country]uint64{"de": 1}}, s.tally(1, 1))
}

func (s *StoreContractSuite) TestATakeAfterTheLastSeasonOrThatDoesNotCountIsPassed() {
	late := EntryOf(s.next, Take{Account: contractAccount(1), Country: "fr", At: contractAt.Add(60 * 24 * time.Hour)}, false)
	s.next++

	s.count(s.batch(late, s.entry(0, 1, "", false), s.entry(0, 1, "fr", true)))

	s.True(s.tally(0, 1).Empty())
	s.True(s.tally(1, 1).Empty())
	s.Equal(Position(3), s.position())
}

func (s *StoreContractSuite) TestNothingIsCountedBeforeTheStart() {
	_, err := s.store.Position(s.T().Context())
	s.Require().ErrorIs(err, ErrNotStarted)
	s.Require().ErrorIs(s.store.Count(s.T().Context(), s.batch(s.entry(0, 1, "fr", false)), contractSeasons), ErrNotStarted)
	_, err = s.store.Rewind(s.T().Context())
	s.Require().ErrorIs(err, ErrNotStarted)

	s.True(s.tally(0, 1).Empty())
}

func (s *StoreContractSuite) TestTheCountBeginsAtItsStartOnce() {
	s.Require().NoError(s.store.Begin(s.T().Context(), 5))
	s.Equal(Position(5), s.position())

	s.Require().ErrorIs(s.store.Begin(s.T().Context(), 9), ErrStarted)
	s.Equal(Position(5), s.position())
}

func (s *StoreContractSuite) TestEntriesThatAreNotTakesMoveThePositionPastThem() {
	s.begin()
	batch, err := BatchOf(0, 5, []Entry{EntryOf(2, Take{Account: contractAccount(1), Country: "fr", At: contractAt}, false)})
	s.Require().NoError(err)

	s.Require().NoError(s.store.Count(s.T().Context(), batch, contractSeasons))

	s.Equal(map[Country]uint64{"fr": 1}, s.tally(0, 1).Tiles)
	s.Equal(Position(5), s.position())
}

func (s *StoreContractSuite) TestABatchCountedTwiceCountsOnce() {
	batch := s.batch(s.entry(0, 1, "fr", false))
	s.count(batch)

	s.Require().ErrorIs(s.store.Count(s.T().Context(), batch, contractSeasons), ErrMoved)

	s.Equal(map[Country]uint64{"fr": 1}, s.tally(0, 1).Tiles)
	s.Equal(Position(1), s.position())
}

func (s *StoreContractSuite) TestARewindForgetsEverySeasonAndStartsAtTheFirstPosition() {
	s.take(0, 1, "fr", 2)
	s.take(1, 2, "de", 1)

	start, err := s.store.Rewind(s.T().Context())

	s.Require().NoError(err)
	s.Equal(Position(0), start)
	s.Equal(Position(0), s.position())
	s.True(s.tally(0, 1).Empty())
	s.True(s.tally(1, 2).Empty())
}

func (s *StoreContractSuite) TestCountingTheSameTakesAgainAfterARewindGivesTheSameStandingsLessWhatWasReverted() {
	first := s.batch(s.entry(0, 1, "fr", false), s.entry(0, 1, "fr", false), s.entry(0, 2, "de", false))
	s.count(first)
	live := []Tally{s.tally(0, 1), s.tally(0, 2)}

	_, err := s.store.Rewind(s.T().Context())
	s.Require().NoError(err)
	s.next = 0
	s.count(first)
	rebuilt := []Tally{s.tally(0, 1), s.tally(0, 2)}
	s.Equal(live, rebuilt)

	_, err = s.store.Rewind(s.T().Context())
	s.Require().NoError(err)
	s.next = 0
	s.count(s.batch(s.entry(0, 1, "fr", false), s.entry(0, 1, "fr", true), s.entry(0, 2, "de", false)))
	s.Equal(map[Country]uint64{"fr": 1}, s.tally(0, 1).Tiles, "the take reverted since is not counted again")
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

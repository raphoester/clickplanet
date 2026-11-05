//go:build testing

package books

import (
	"time"

	"github.com/stretchr/testify/suite"
)

var (
	contractStart   = time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	contractEnd     = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	contractBook    = NewBook("season-0", contractStart)
	contractNext    = NewBook("season-1", contractEnd.Add(24*time.Hour))
	contractDay     = Round{ID: "season-0/day-24", Book: contractBook.ID, Times: 1, StartsAt: contractEnd.Add(-26 * time.Hour)}
	contractBattle  = Round{ID: "0192f5a0-7c00-7000-8000-000000000031", Book: contractBook.ID, Times: 3, StartsAt: contractEnd.Add(-2 * time.Hour)}
	contractNextDay = Round{ID: "season-1/day-1", Book: contractNext.ID, Times: 1, StartsAt: contractNext.StartsAt}
)

type BooksContractSuite struct {
	suite.Suite

	NewStore func() Books

	store Books
}

func (s *BooksContractSuite) SetupTest() {
	s.store = s.NewStore()
}

func (s *BooksContractSuite) open(book Book) bool {
	opened, err := s.store.Open(s.T().Context(), book)
	s.Require().NoError(err)
	return opened
}

func (s *BooksContractSuite) find(id BookID) (Book, bool) {
	book, found, err := s.store.Find(s.T().Context(), id)
	s.Require().NoError(err)
	return book, found
}

func (s *BooksContractSuite) TestABookNeverOpenedIsNotFound() {
	_, found := s.find(contractBook.ID)
	s.False(found)
}

func (s *BooksContractSuite) TestABookOpenedIsFoundOpenAndOpeningItAgainChangesNothing() {
	s.True(s.open(contractBook))
	s.False(s.open(NewBook(contractBook.ID, contractEnd)))

	book, found := s.find(contractBook.ID)
	s.True(found)
	s.Equal(contractBook, book)
}

func (s *BooksContractSuite) TestABookClosedKeepsItsEndAndTiebreakAndIsClosedOnce() {
	s.open(contractBook)
	closed := contractBook.ClosedAt(contractEnd, contractBattle.ID)

	first, err := s.store.Close(s.T().Context(), closed)
	s.Require().NoError(err)
	again, err := s.store.Close(s.T().Context(), contractBook.ClosedAt(contractEnd.Add(time.Hour), ""))
	s.Require().NoError(err)

	s.True(first)
	s.False(again, "only the first close closes the book")
	book, _ := s.find(contractBook.ID)
	s.Equal(closed, book)
}

func (s *BooksContractSuite) TestABookClosedWithNoTiebreakHasNone() {
	s.open(contractBook)
	closed := contractBook.ClosedAt(contractEnd, "")

	_, err := s.store.Close(s.T().Context(), closed)
	s.Require().NoError(err)

	book, _ := s.find(contractBook.ID)
	s.Equal(closed, book)
}

func (s *BooksContractSuite) TestEveryBookIsListedTheFirstStartFirst() {
	s.open(contractNext)
	s.open(contractBook)

	all, err := s.store.All(s.T().Context())
	s.Require().NoError(err)
	s.Equal([]Book{contractBook, contractNext}, all)
}

// RoundsContractSuite needs the books of its rounds open: NewStores answers both stores over the same data.
type RoundsContractSuite struct {
	suite.Suite

	NewStores func() (Books, Rounds)

	books  Books
	rounds Rounds
}

func (s *RoundsContractSuite) SetupTest() {
	s.books, s.rounds = s.NewStores()
	for _, book := range []Book{contractBook, contractNext} {
		_, err := s.books.Open(s.T().Context(), book)
		s.Require().NoError(err)
	}
}

func (s *RoundsContractSuite) open(rounds ...Round) {
	for _, round := range rounds {
		_, err := s.rounds.Open(s.T().Context(), round)
		s.Require().NoError(err)
	}
}

func (s *RoundsContractSuite) find(id RoundID) (Round, bool) {
	round, found, err := s.rounds.Find(s.T().Context(), id)
	s.Require().NoError(err)
	return round, found
}

func (s *RoundsContractSuite) score(scored Scored) bool {
	done, err := s.rounds.Score(s.T().Context(), scored)
	s.Require().NoError(err)
	return done
}

func (s *RoundsContractSuite) scoredIn(book BookID) []Scored {
	scored, err := s.rounds.ScoredIn(s.T().Context(), book)
	s.Require().NoError(err)
	return scored
}

func (s *RoundsContractSuite) TestARoundNeverOpenedIsNotFound() {
	_, found := s.find(contractDay.ID)
	s.False(found)
}

func (s *RoundsContractSuite) TestARoundOpenedIsFoundOpenAndOpeningItAgainChangesNothing() {
	opened, err := s.rounds.Open(s.T().Context(), contractDay)
	s.Require().NoError(err)
	again, err := s.rounds.Open(s.T().Context(), Round{ID: contractDay.ID, Book: contractBook.ID, Times: 2, StartsAt: contractEnd})
	s.Require().NoError(err)

	s.True(opened)
	s.False(again)
	round, found := s.find(contractDay.ID)
	s.True(found)
	s.Equal(contractDay, round)
}

func (s *RoundsContractSuite) TestARoundScoredKeepsItsEndAndResultsAndIsScoredOnce() {
	s.open(contractDay)
	results := []Result{{Country: "fr", Rank: 1, Points: 25, Share: 0.3}, {Country: "de", Rank: 2, Points: 18, Share: 0.125}}
	scored := Scored{Round: contractDay, Results: results}
	scored.Round.Scored, scored.Round.EndsAt = true, contractEnd.Add(-2*time.Hour)

	s.True(s.score(scored))
	s.False(s.score(Scored{Round: scored.Round, Results: []Result{{Country: "de", Rank: 1, Points: 25, Share: 0.5}}}),
		"only the first score scores the round")

	round, _ := s.find(contractDay.ID)
	s.Equal(scored.Round, round)
	s.Equal([]Scored{scored}, s.scoredIn(contractBook.ID))
}

func (s *RoundsContractSuite) TestABookListsTheRoundsItScoredInTheOrderTheyEnded() {
	s.open(contractBattle, contractDay, contractNextDay)
	battle := contractBattle.ScoredAt(contractEnd, Snapshot{Tiles: 10, Held: map[Country]uint32{"de": 4, "fr": 1}})
	day := contractDay.ScoredAt(contractEnd.Add(-2*time.Hour), Snapshot{Tiles: 10, Held: map[Country]uint32{"fr": 3}})
	none := contractNextDay.ScoredAt(contractNext.StartsAt.Add(24*time.Hour), Snapshot{Tiles: 10})
	s.score(battle)
	s.score(day)
	s.score(none)

	s.Equal([]Scored{day, battle}, s.scoredIn(contractBook.ID))
	s.Equal([]Scored{{Round: none.Round}}, s.scoredIn(contractNext.ID), "a round scored with no results is listed with none")
}

func (s *RoundsContractSuite) TestABookWithNoRoundScoredListsNone() {
	s.open(contractDay)

	s.Empty(s.scoredIn(contractBook.ID))
}

func (s *RoundsContractSuite) TestTheRoundsOfBooksAreListedTheFirstStartFirst() {
	s.open(contractNextDay, contractBattle, contractDay)
	scored := contractDay.ScoredAt(contractEnd.Add(-2*time.Hour), Snapshot{Tiles: 1})
	s.score(scored)

	of, err := s.rounds.Of(s.T().Context(), []BookID{contractBook.ID})
	s.Require().NoError(err)
	s.Equal([]Round{scored.Round, contractBattle}, of)

	of, err = s.rounds.Of(s.T().Context(), []BookID{contractBook.ID, contractNext.ID})
	s.Require().NoError(err)
	s.Equal([]Round{scored.Round, contractBattle, contractNextDay}, of)

	of, err = s.rounds.Of(s.T().Context(), nil)
	s.Require().NoError(err)
	s.Empty(of)
}

// TakesContractSuite pins what a round counts of its takes. CountedIn reads it back, since the write side reads
// nothing.
type TakesContractSuite struct {
	suite.Suite

	NewStores func() (Books, Rounds, Takes)
	CountedIn func(round RoundID) Counted

	takes Takes
}

// Counted is the tiles a round counted for each flag, and for each account and flag.
type Counted struct {
	Countries map[Country]uint64
	Players   map[AccountID]map[Country]uint64
}

var (
	contractPlayer = AccountID{1}
	contractOther  = AccountID{2}
)

func (s *TakesContractSuite) SetupTest() {
	var (
		books  Books
		rounds Rounds
	)
	books, rounds, s.takes = s.NewStores()
	_, err := books.Open(s.T().Context(), contractBook)
	s.Require().NoError(err)
	for _, round := range []Round{contractDay, contractBattle} {
		_, err := rounds.Open(s.T().Context(), round)
		s.Require().NoError(err)
	}
}

func (s *TakesContractSuite) count(rounds []RoundID, takes ...Take) {
	for _, take := range takes {
		s.Require().NoError(s.takes.Count(s.T().Context(), rounds, take))
	}
}

func (s *TakesContractSuite) TestARoundCountsEachTakeOnce() {
	at := contractBattle.StartsAt
	both := []RoundID{contractDay.ID, contractBattle.ID}

	s.count(
		both,
		Take{Account: contractPlayer, Country: "fr", At: at, Position: 2},
		Take{Account: contractPlayer, Country: "fr", At: at, Position: 3},
		Take{Account: contractPlayer, Country: "fr", At: at, Position: 3},
		Take{Account: contractPlayer, Country: "fr", At: at, Position: 1},
		Take{Account: contractPlayer, Country: "be", At: at, Position: 4},
		Take{Account: contractOther, Country: "de", At: at, Position: 5},
	)
	s.count([]RoundID{contractDay.ID}, Take{Account: contractOther, Country: "de", At: at, Position: 6})

	s.Equal(Counted{
		Countries: map[Country]uint64{"fr": 2, "be": 1, "de": 1},
		Players: map[AccountID]map[Country]uint64{
			contractPlayer: {"fr": 2, "be": 1},
			contractOther:  {"de": 1},
		},
	}, s.CountedIn(contractBattle.ID), "a take told again, or before the last counted, once")
	s.Equal(map[Country]uint64{"fr": 2, "be": 1, "de": 2}, s.CountedIn(contractDay.ID).Countries)
}

func (s *TakesContractSuite) TestARoundCountsEachTakeOfASpreadOnce() {
	at := contractBattle.StartsAt
	spread := []Take{
		{Account: contractPlayer, Country: "fr", At: at, Position: 4},
		{Account: contractPlayer, Country: "fr", At: at, Position: 5, Index: 0},
		{Account: contractPlayer, Country: "fr", At: at, Position: 5, Index: 1},
		{Account: contractPlayer, Country: "fr", At: at, Position: 5, Index: 2},
	}

	s.count([]RoundID{contractBattle.ID}, spread...)
	s.count([]RoundID{contractBattle.ID}, spread...)

	s.Equal(map[Country]uint64{"fr": 4}, s.CountedIn(contractBattle.ID).Countries)
}

func (s *TakesContractSuite) TestATakeInNoRoundCountsNothing() {
	s.count(nil, Take{Account: contractPlayer, Country: "fr", At: contractBattle.StartsAt, Position: 1})

	s.Empty(s.CountedIn(contractDay.ID).Countries)
}

func (s *TakesContractSuite) TestAnAccountDeletedIsForgottenAndItsFlagKeepsItsTiles() {
	both := []RoundID{contractBattle.ID}
	s.count(
		both,
		Take{Account: contractPlayer, Country: "fr", At: contractBattle.StartsAt, Position: 1},
		Take{Account: contractOther, Country: "fr", At: contractBattle.StartsAt, Position: 2},
	)

	s.Require().NoError(s.takes.ForgetAccount(s.T().Context(), contractPlayer))

	s.Equal(Counted{
		Countries: map[Country]uint64{"fr": 2},
		Players:   map[AccountID]map[Country]uint64{contractOther: {"fr": 1}},
	}, s.CountedIn(contractBattle.ID))
}

type GrantsContractSuite struct {
	suite.Suite

	NewStore func() Grants

	store Grants
}

func (s *GrantsContractSuite) SetupTest() {
	s.store = s.NewStore()
}

func (s *GrantsContractSuite) keep(grant Grant) bool {
	kept, err := s.store.Keep(s.T().Context(), grant)
	s.Require().NoError(err)
	return kept
}

func (s *GrantsContractSuite) since(at time.Time) []Grant {
	grants, err := s.store.Since(s.T().Context(), at)
	s.Require().NoError(err)
	return grants
}

var contractGrant = Grant{ID: "0192f5a0-7c00-7000-8000-0000000000a1", Country: "fr", Points: 2, At: contractEnd.Add(-3 * time.Hour)}

func (s *GrantsContractSuite) TestAGrantIsKeptOnce() {
	s.True(s.keep(contractGrant))
	s.False(s.keep(Grant{ID: contractGrant.ID, Country: "de", Points: 3, At: contractEnd}))

	s.Equal([]Grant{contractGrant}, s.since(contractStart))
}

func (s *GrantsContractSuite) TestTheGrantsFromATimeAreListedInTheOrderGranted() {
	later := Grant{ID: "later", Country: "de", Points: 1, At: contractEnd}
	earlier := Grant{ID: "earlier", Country: "it", Points: 3, At: contractStart}
	for _, grant := range []Grant{later, contractGrant, earlier} {
		s.keep(grant)
	}

	s.Equal([]Grant{earlier, contractGrant, later}, s.since(contractStart))
	s.Equal([]Grant{later}, s.since(contractEnd), "from the time on")
}

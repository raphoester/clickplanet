package board_query_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/departures"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/departures/postgres_departure_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/players/postgres_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_board_handler/board_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings/postgres_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db    *cppg.Postgres
	store *postgres_contribution_store.Store
	taken uint64
}

func (s *testSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "scoring", migrations.FS)
	s.store = postgres_contribution_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func account(n int) standings.AccountID {
	return standings.AccountID{0: 0x01, 14: byte(n >> 8), 15: byte(n)}
}

func (s *testSuite) take(book books.BookID, n int, country standings.Country, tiles int) {
	takes := make([]standings.Take, 0, tiles)
	for range tiles {
		s.taken++
		takes = append(takes, standings.Take{Account: account(n), Country: country, Position: s.taken})
	}
	s.Require().NoError(s.store.RecordTakes(s.T().Context(), book, account(n), takes))
}

func (s *testSuite) named(n int) {
	s.Require().NoError(postgres_player_store.New(s.db).Keep(s.T().Context(), account(n)))
}

func (s *testSuite) player(n int, country standings.Country, tiles int) {
	s.take(seasonZero, n, country, tiles)
	s.named(n)
}

func (s *testSuite) guest(n int, country standings.Country, tiles int) {
	s.take(seasonZero, n, country, tiles)
}

func (s *testSuite) decided(n int, decision departures.Decision) {
	s.Require().NoError(postgres_departure_store.New(s.db).Change(s.T().Context(), account(n),
		func(departure departures.Departure) departures.Departure { return departure.Decided(decision) }))
}

const (
	seasonZero = books.BookID("season-0")
	seasonOne  = books.BookID("season-1")
)

func (s *testSuite) query() *board_query.PostgresQuery {
	return board_query.NewPostgresQuery(s.db, cpcountries.New())
}

func (s *testSuite) standings(country string) *scoringv1.GetBoardResponse {
	return s.of(seasonZero, country)
}

func (s *testSuite) of(book books.BookID, country string) *scoringv1.GetBoardResponse {
	res, err := s.query().Board(s.T().Context(), board_query.View{Book: book, Country: country})
	s.Require().NoError(err)
	return res
}

func lines(res *scoringv1.GetBoardResponse) []string {
	shown := make([]string, len(res.GetStandings()))
	for i, standing := range res.GetStandings() {
		shown[i] = fmt.Sprintf("%d %s %s %d", standing.GetRank(), nameOf(standing.GetAccountId()), standing.GetCountryId(), standing.GetTiles())
	}
	return shown
}

func nameOf(id string) string {
	for n := range 2000 {
		if account(n).String() == id {
			return fmt.Sprintf("player_%d", n)
		}
	}
	return id
}

func (s *testSuite) TestTheTopIsTheTenBestPlayersWithAUsername() {
	for n := 1; n <= 12; n++ {
		s.player(n, "fr", 20-n)
	}
	s.guest(100, "fr", 30)
	s.guest(101, "de", 15)

	res := s.standings("")

	s.Require().Len(res.GetStandings(), board_query.Shown)
	s.Equal("1 player_1 fr 19", lines(res)[0])
	s.Equal("10 player_10 fr 10", lines(res)[9])
	s.Equal(account(1).String(), res.GetStandings()[0].GetAccountId())
	s.Equal(account(10).String(), res.GetStandings()[9].GetAccountId())
}

func (s *testSuite) TestPlayersWithAsManyTilesShareARank() {
	s.player(1, "fr", 5)
	s.player(2, "de", 5)
	s.guest(3, "fr", 4)
	s.player(4, "it", 3)

	s.Equal([]string{"1 player_1 fr 5", "1 player_2 de 5", "3 player_4 it 3"}, lines(s.standings("")))
}

func (s *testSuite) TestTheTopOfACountryRanksEveryPlayerByTheTilesTakenForIt() {
	s.player(1, "de", 9)
	s.take(seasonZero, 1, "fr", 8)
	s.player(2, "fr", 2)
	s.player(3, "fr", 4)
	s.guest(4, "fr", 5)
	s.take(seasonZero, 3, "it", 1)

	fr := s.standings("fr")
	s.Equal([]string{"1 player_1 fr 8", "2 player_3 fr 4", "3 player_2 fr 2"}, lines(fr))
	s.Equal(account(3).String(), fr.GetStandings()[1].GetAccountId())
	s.Equal([]string{"1 player_1 de 9"}, lines(s.standings("de")))
	s.Equal([]string{"1 player_3 it 1"}, lines(s.standings("it")))
}

func (s *testSuite) TestTheTopOfTheWholeMapCountsOnlyTheMainFlag() {
	s.player(1, "de", 9)
	s.take(seasonZero, 1, "fr", 8)
	s.player(2, "fr", 10)

	s.Equal([]string{"1 player_2 fr 10", "2 player_1 de 9"}, lines(s.standings("")))
}

func (s *testSuite) TestAnAccountAskedDeletedIsNotRankedUntilItReturns() {
	s.player(1, "fr", 9)
	s.player(2, "fr", 2)
	s.decided(1, departures.Leaves(time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)))

	s.Equal([]string{"1 player_2 fr 2"}, lines(s.standings("")))
	s.Equal([]string{"1 player_2 fr 2"}, lines(s.standings("fr")))

	s.decided(1, departures.Returns(time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)))

	s.Equal([]string{"1 player_1 fr 9", "2 player_2 fr 2"}, lines(s.standings("")))
}

func (s *testSuite) TestTheTopReadsPastAPageOfGuests() {
	for n := range 250 {
		s.guest(1000+n, "fr", 2)
	}
	s.player(1, "fr", 1)

	s.Equal([]string{"1 player_1 fr 1"}, lines(s.standings("")))
}

func (s *testSuite) TestAnEmptyBookIsAnEmptyTop() {
	s.Empty(s.standings("").GetStandings())
}

func (s *testSuite) TestOnlyTheBookAskedCounts() {
	s.player(1, "fr", 3)
	s.take(seasonOne, 2, "de", 1)
	s.named(2)

	s.Equal([]string{"1 player_1 fr 3"}, lines(s.of(seasonZero, "")))
	s.Equal([]string{"1 player_2 de 1"}, lines(s.of(seasonOne, "")))
	s.Empty(s.of("season-2", "").GetStandings())
}

func (s *testSuite) TestACountryThatIsNotOneIsRefusedAndReadsNothing() {
	s.player(1, "fr", 3)

	_, err := s.query().Board(s.T().Context(), board_query.View{Book: seasonZero, Country: "zz"})

	s.Require().ErrorIs(err, board_query.ErrUnknownCountry)
}

func (s *testSuite) TestABoardIsTheTopOfTheStandings() {
	s.player(1, "fr", 5)
	s.guest(2, "fr", 9)
	s.player(3, "de", 7)

	board, found, err := s.query().Live(s.T().Context(), board_query.View{Book: seasonZero})

	s.Require().NoError(err)
	s.True(found)
	s.Equal(string(seasonZero), board.GetBookId())
	s.Equal(lines(s.standings("")), lines(&scoringv1.GetBoardResponse{Standings: board.GetStandings()}))
}

func (s *testSuite) TestABoardOfACountryThatIsNotOneIsRefused() {
	_, _, err := s.query().Live(s.T().Context(), board_query.View{Book: seasonZero, Country: "zz"})

	s.Require().ErrorIs(err, board_query.ErrUnknownCountry)
}

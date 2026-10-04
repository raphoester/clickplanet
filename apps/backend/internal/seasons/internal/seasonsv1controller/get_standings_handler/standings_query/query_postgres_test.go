package standings_query_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_standings_handler/standings_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/postgres_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type fakeAuthors struct {
	named map[standings.AccountID]*playerv1.Author
	asked int
	err   error
}

func (f *fakeAuthors) Authors(
	_ context.Context,
	accounts []standings.AccountID,
) (map[standings.AccountID]*playerv1.Author, error) {
	f.asked++
	if f.err != nil {
		return nil, f.err
	}
	found := make(map[standings.AccountID]*playerv1.Author, len(accounts))
	for _, account := range accounts {
		if author, known := f.named[account]; known {
			found[account] = author
		}
	}
	return found, nil
}

var seasonZeroEnds = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)

func twoSeasons() calendar.Calendar {
	return calendar.New(calendar.Config{List: []calendar.Entry{
		{Number: 0, EndsAt: seasonZeroEnds, Finale: 2 * time.Hour},
		{Number: 1, EndsAt: seasonZeroEnds.AddDate(0, 2, 0), Finale: 2 * time.Hour},
	}})
}

type testSuite struct {
	suite.Suite

	db      *cppg.Postgres
	store   *postgres_contribution_store.Store
	authors *fakeAuthors
	clock   *cptime.FixedClock
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "seasons", migrations.FS)
	s.store = postgres_contribution_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.authors = &fakeAuthors{named: map[standings.AccountID]*playerv1.Author{}}
	s.clock = cptime.NewFixedClock(seasonZeroEnds.Add(-24 * time.Hour))
}

func account(n int) standings.AccountID {
	return standings.AccountID{0: 0x01, 14: byte(n >> 8), 15: byte(n)}
}

func (s *testSuite) take(season calendar.Number, n int, country standings.Country, tiles int) {
	for range tiles {
		s.Require().NoError(s.store.RecordTake(s.T().Context(), season, standings.Take{Account: account(n), Country: country}))
	}
}

func (s *testSuite) player(n int, country standings.Country, tiles int) {
	s.take(0, n, country, tiles)
	s.authors.named[account(n)] = &playerv1.Author{
		AccountId: account(n).String(), Name: fmt.Sprintf("player_%d", n), Color: playerv1.NameColor(n % 13),
	}
}

func (s *testSuite) guest(n int, country standings.Country, tiles int) {
	s.take(0, n, country, tiles)
	s.authors.named[account(n)] = &playerv1.Author{AccountId: account(n).String(), Name: fmt.Sprintf("guest_%06x", n), Guest: true}
}

func (s *testSuite) query() *standings_query.PostgresQuery {
	return standings_query.NewPostgresQuery(s.db, s.authors, twoSeasons(), s.clock, cpcountries.New())
}

func (s *testSuite) standings(country string) *seasonsv1.GetStandingsResponse {
	res, err := s.query().Standings(s.T().Context(), country)
	s.Require().NoError(err)
	return res
}

func lines(res *seasonsv1.GetStandingsResponse) []string {
	shown := make([]string, len(res.GetStandings()))
	for i, standing := range res.GetStandings() {
		shown[i] = fmt.Sprintf("%d %s %s %d", standing.GetRank(), standing.GetName(), standing.GetCountryId(), standing.GetTiles())
	}
	return shown
}

func (s *testSuite) TestTheTopIsTheTenBestPlayersWithAUsername() {
	for n := 1; n <= 12; n++ {
		s.player(n, "fr", 20-n)
	}
	s.guest(100, "fr", 30)
	s.guest(101, "de", 15)

	res := s.standings("")

	s.Require().Len(res.GetStandings(), standings_query.Shown)
	s.Equal("1 player_1 fr 19", lines(res)[0])
	s.Equal("10 player_10 fr 10", lines(res)[9])
	s.Equal(playerv1.NameColor_NAME_COLOR_RED, res.GetStandings()[0].GetColor())
}

func (s *testSuite) TestPlayersWithAsManyTilesShareARank() {
	s.player(1, "fr", 5)
	s.player(2, "de", 5)
	s.guest(3, "fr", 4)
	s.player(4, "it", 3)

	s.Equal([]string{"1 player_1 fr 5", "1 player_2 de 5", "3 player_4 it 3"}, lines(s.standings("")))
}

func (s *testSuite) TestEachStandingWearsTheTitleItsPlayerWears() {
	warmaster := &playerv1.Title{
		Id: "warmaster", Name: "Warmaster",
		Rank: &playerv1.Rank{TrackId: "conquest", TrackName: "Conquest", Number: 5, Count: 5},
	}
	s.player(1, "fr", 5)
	s.authors.named[account(1)].WornTitle = warmaster
	s.player(2, "de", 3)

	res := s.standings("")

	s.Require().Len(res.GetStandings(), 2)
	s.True(proto.Equal(warmaster, res.GetStandings()[0].GetWornTitle()), res.GetStandings()[0].GetWornTitle())
	s.Nil(res.GetStandings()[1].GetWornTitle(), "a player who wears no title wears none here")
}

func (s *testSuite) TestTheTopOfACountryRanksEveryPlayerByTheTilesTakenForIt() {
	s.player(1, "de", 9)
	s.take(0, 1, "fr", 8)
	s.player(2, "fr", 2)
	s.player(3, "fr", 4)
	s.guest(4, "fr", 5)
	s.take(0, 3, "it", 1)

	s.Equal([]string{"1 player_1 fr 8", "2 player_3 fr 4", "3 player_2 fr 2"}, lines(s.standings("fr")))
	s.Equal([]string{"1 player_1 de 9"}, lines(s.standings("de")))
	s.Equal([]string{"1 player_3 it 1"}, lines(s.standings("it")))
}

func (s *testSuite) TestTheTopOfTheWholeMapCountsOnlyTheMainFlag() {
	s.player(1, "de", 9)
	s.take(0, 1, "fr", 8)
	s.player(2, "fr", 10)

	s.Equal([]string{"1 player_2 fr 10", "2 player_1 de 9"}, lines(s.standings("")))
}

func (s *testSuite) TestAnAccountThePlayerModuleCannotNameIsNotRanked() {
	s.take(0, 1, "fr", 9)
	s.player(2, "fr", 2)

	s.Equal([]string{"1 player_2 fr 2"}, lines(s.standings("")))
}

func (s *testSuite) TestTheTopReadsPastAPageOfGuests() {
	for n := range 250 {
		s.guest(1000+n, "fr", 2)
	}
	s.player(1, "fr", 1)

	s.Equal([]string{"1 player_1 fr 1"}, lines(s.standings("")))
	s.Equal(2, s.authors.asked)
}

func (s *testSuite) TestAnEmptySeasonIsAnEmptyTopAndAsksNobody() {
	s.Empty(s.standings("").GetStandings())
	s.Zero(s.authors.asked)
}

func (s *testSuite) TestOnlyTheCurrentSeasonCounts() {
	s.player(1, "fr", 3)
	s.take(1, 2, "de", 1)
	s.authors.named[account(2)] = &playerv1.Author{AccountId: account(2).String(), Name: "player_2"}

	s.Equal([]string{"1 player_1 fr 3"}, lines(s.standings("")))

	s.clock.Advance(48 * time.Hour)
	s.Equal([]string{"1 player_2 de 1"}, lines(s.standings("")))
}

func (s *testSuite) TestNoSeasonIsAnEmptyAnswer() {
	s.player(1, "fr", 3)
	s.clock.Advance(365 * 24 * time.Hour)

	s.Empty(s.standings("").GetStandings())
}

func (s *testSuite) TestACountryThatIsNotOneIsRefusedAndReadsNothing() {
	s.player(1, "fr", 3)

	_, err := s.query().Standings(s.T().Context(), "zz")

	s.Require().ErrorIs(err, standings_query.ErrUnknownCountry)
	s.Zero(s.authors.asked)
}

func (s *testSuite) TestABoardIsTheTopAndTheAccountOfEachLineInItsOrder() {
	s.player(1, "fr", 5)
	s.guest(2, "fr", 9)
	s.player(3, "de", 7)

	board, accounts, err := s.query().Board(s.T().Context(), "")

	s.Require().NoError(err)
	s.Equal(lines(s.standings("")), lines(&seasonsv1.GetStandingsResponse{Standings: board.GetStandings()}))
	s.Equal([]standings.AccountID{account(3), account(1)}, accounts)
}

func (s *testSuite) TestABoardOfACountryThatIsNotOneIsRefused() {
	_, _, err := s.query().Board(s.T().Context(), "zz")

	s.Require().ErrorIs(err, standings_query.ErrUnknownCountry)
}

func (s *testSuite) TestAFailureToNameIsAnError() {
	s.player(1, "fr", 3)
	refused := errors.New("the player module is down")
	s.authors.err = refused

	_, err := s.query().Standings(s.T().Context(), "")

	s.Require().ErrorIs(err, refused)
}

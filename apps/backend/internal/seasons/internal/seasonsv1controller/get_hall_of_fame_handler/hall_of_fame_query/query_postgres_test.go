package hall_of_fame_query_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/postgres_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_hall_of_fame_handler/hall_of_fame_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/postgres_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
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

func utc(month time.Month, day, hour int) time.Time {
	return time.Date(2026, month, day, hour, 0, 0, 0, time.UTC)
}

var (
	thirtieth   = rounds.Round{EndsAt: utc(10, 30, 21)}
	thirtyFirst = rounds.Round{EndsAt: utc(10, 31, 21)}
	finale      = rounds.Round{EndsAt: utc(10, 31, 23), Finale: true}
	nextFinale  = rounds.Round{Season: 1, EndsAt: utc(12, 31, 23), Finale: true}
)

type testSuite struct {
	suite.Suite

	db            *cppg.Postgres
	rounds        *postgres_round_store.Store
	contributions *postgres_contribution_store.Store
	authors       *fakeAuthors
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "seasons", migrations.FS)
	s.rounds = postgres_round_store.New(s.db)
	s.contributions = postgres_contribution_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.authors = &fakeAuthors{named: map[standings.AccountID]*playerv1.Author{}}
}

func (s *testSuite) counted(round rounds.Round) {
	s.Require().NoError(s.rounds.RecordSnapshot(s.T().Context(), round, rounds.Snapshot{Tiles: 100, Held: map[rounds.Country]uint32{"fr": 1}}))
}

func (s *testSuite) closed(round rounds.Round, results ...rounds.Result) {
	s.counted(round)
	s.Require().NoError(s.rounds.Close(s.T().Context(), round, results))
}

func account(n int) standings.AccountID {
	return standings.AccountID{0: 0x01, 14: byte(n >> 8), 15: byte(n)}
}

func (s *testSuite) take(season calendar.Number, n int, country standings.Country, tiles int) {
	for range tiles {
		s.Require().NoError(s.contributions.RecordTake(s.T().Context(), season, standings.Take{Account: account(n), Country: country}))
	}
}

func (s *testSuite) player(season calendar.Number, n int, country standings.Country, tiles int) {
	s.take(season, n, country, tiles)
	s.authors.named[account(n)] = &playerv1.Author{
		AccountId: account(n).String(), Name: fmt.Sprintf("player_%d", n), Color: playerv1.NameColor(n % 13),
	}
}

func (s *testSuite) guest(season calendar.Number, n int, country standings.Country, tiles int) {
	s.take(season, n, country, tiles)
	s.authors.named[account(n)] = &playerv1.Author{AccountId: account(n).String(), Name: fmt.Sprintf("guest_%06x", n), Guest: true}
}

func (s *testSuite) hall() []*seasonsv1.EndedSeason {
	res, err := hall_of_fame_query.NewPostgresQuery(s.db, s.authors).HallOfFame(s.T().Context())
	s.Require().NoError(err)
	return res.GetSeasons()
}

func (s *testSuite) onlySeason() *seasonsv1.EndedSeason {
	hall := s.hall()
	s.Require().Len(hall, 1)
	return hall[0]
}

func lines(standings []*seasonsv1.Standing) []string {
	shown := make([]string, len(standings))
	for i, standing := range standings {
		shown[i] = fmt.Sprintf("%d %s %s %d", standing.GetRank(), standing.GetName(), standing.GetCountryId(), standing.GetTiles())
	}
	return shown
}

func (s *testSuite) equalScores(want, got []*seasonsv1.Score) {
	s.Require().Len(got, len(want), got)
	for i := range want {
		s.True(proto.Equal(want[i], got[i]), got[i])
	}
}

func (s *testSuite) TestBeforeAFinaleClosesTheHallIsEmptyAndAsksNobody() {
	s.closed(thirtieth, rounds.Result{Country: "fr", Rank: 1, Points: 25})
	s.counted(finale)
	s.player(0, 1, "fr", 3)

	s.Empty(s.hall())
	s.Zero(s.authors.asked)
}

func (s *testSuite) TestASeasonWhoseFinaleClosedIsInTheHallWithItsEnd() {
	s.closed(finale, rounds.Result{Country: "fr", Rank: 1, Points: 75})

	season := s.onlySeason()

	s.Equal(uint32(0), season.GetNumber())
	s.Equal(finale.EndsAt.UnixMilli(), season.GetEndedAtUnixMs())
}

func (s *testSuite) TestTheTableIsEveryCountryWithPointsChampionFirst() {
	s.closed(thirtieth,
		rounds.Result{Country: "de", Rank: 1, Points: 25},
		rounds.Result{Country: "fr", Rank: 2, Points: 18},
		rounds.Result{Country: "es", Rank: 11, Points: 0})
	s.closed(thirtyFirst,
		rounds.Result{Country: "fr", Rank: 1, Points: 25},
		rounds.Result{Country: "de", Rank: 2, Points: 18})
	s.closed(finale,
		rounds.Result{Country: "fr", Rank: 1, Points: 75},
		rounds.Result{Country: "it", Rank: 2, Points: 54})

	s.equalScores([]*seasonsv1.Score{
		{Rank: 1, CountryId: "fr", Points: 118, RoundsWon: 2},
		{Rank: 2, CountryId: "it", Points: 54},
		{Rank: 3, CountryId: "de", Points: 43, RoundsWon: 1},
	}, s.onlySeason().GetScores())
}

func (s *testSuite) TestCountriesLevelOnPointsAreRankedByTheRoundsTheyWon() {
	s.closed(thirtieth,
		rounds.Result{Country: "de", Rank: 1, Points: 25},
		rounds.Result{Country: "fr", Rank: 2, Points: 18})
	s.closed(thirtyFirst,
		rounds.Result{Country: "de", Rank: 1, Points: 25},
		rounds.Result{Country: "fr", Rank: 9, Points: 2})
	s.closed(finale,
		rounds.Result{Country: "fr", Rank: 1, Points: 75},
		rounds.Result{Country: "de", Rank: 3, Points: 45})

	s.equalScores([]*seasonsv1.Score{
		{Rank: 1, CountryId: "de", Points: 95, RoundsWon: 2},
		{Rank: 2, CountryId: "fr", Points: 95, RoundsWon: 1},
	}, s.onlySeason().GetScores())
}

func (s *testSuite) TestCountriesLevelOnEverythingShareTheTitle() {
	s.closed(thirtieth,
		rounds.Result{Country: "de", Rank: 1, Points: 25},
		rounds.Result{Country: "fr", Rank: 1, Points: 25},
		rounds.Result{Country: "it", Rank: 3, Points: 15})
	s.closed(finale,
		rounds.Result{Country: "de", Rank: 1, Points: 75},
		rounds.Result{Country: "fr", Rank: 1, Points: 75})

	s.equalScores([]*seasonsv1.Score{
		{Rank: 1, CountryId: "de", Points: 100, RoundsWon: 2},
		{Rank: 1, CountryId: "fr", Points: 100, RoundsWon: 2},
		{Rank: 3, CountryId: "it", Points: 15},
	}, s.onlySeason().GetScores())
}

func (s *testSuite) TestTheTopPlayersAreTheTenBestWithAUsernameByTheirMainFlag() {
	s.closed(finale, rounds.Result{Country: "fr", Rank: 1, Points: 75})
	for n := 1; n <= 12; n++ {
		s.player(0, n, "fr", 20-n)
	}
	s.guest(0, 100, "fr", 30)
	s.take(0, 101, "de", 25)
	s.take(0, 2, "de", 19)

	standings := s.onlySeason().GetStandings()

	s.Require().Len(standings, hall_of_fame_query.Shown)
	s.Equal("1 player_1 fr 19", lines(standings)[0])
	s.Equal("1 player_2 de 19", lines(standings)[1])
	s.Equal("3 player_3 fr 17", lines(standings)[2])
	s.Equal("10 player_10 fr 10", lines(standings)[9])
	s.Equal(account(2).String(), standings[1].GetAccountId())
	s.Equal(playerv1.NameColor(2), standings[1].GetColor())
}

func (s *testSuite) TestEachTopPlayerWearsTheTitleItWearsNow() {
	s.closed(finale, rounds.Result{Country: "fr", Rank: 1, Points: 75})
	warmaster := &playerv1.Title{Id: "warmaster", Name: "Warmaster"}
	s.player(0, 1, "fr", 5)
	s.authors.named[account(1)].WornTitle = warmaster

	standings := s.onlySeason().GetStandings()

	s.Require().Len(standings, 1)
	s.True(proto.Equal(warmaster, standings[0].GetWornTitle()), standings[0].GetWornTitle())
}

func (s *testSuite) TestTheTopReadsPastAPageOfGuests() {
	s.closed(finale, rounds.Result{Country: "fr", Rank: 1, Points: 75})
	for n := range 250 {
		s.guest(0, 1000+n, "fr", 2)
	}
	s.player(0, 1, "fr", 1)

	s.Equal([]string{"1 player_1 fr 1"}, lines(s.onlySeason().GetStandings()))
	s.Equal(2, s.authors.asked)
}

func (s *testSuite) TestEachSeasonEndedIsInTheHallNewestFirstWithItsOwnTableAndPlayers() {
	s.closed(finale, rounds.Result{Country: "fr", Rank: 1, Points: 75})
	s.closed(nextFinale, rounds.Result{Country: "de", Rank: 1, Points: 75})
	s.player(0, 1, "fr", 3)
	s.player(1, 2, "de", 4)

	hall := s.hall()

	s.Require().Len(hall, 2)
	s.Equal(uint32(1), hall[0].GetNumber())
	s.Equal(nextFinale.EndsAt.UnixMilli(), hall[0].GetEndedAtUnixMs())
	s.equalScores([]*seasonsv1.Score{{Rank: 1, CountryId: "de", Points: 75, RoundsWon: 1}}, hall[0].GetScores())
	s.Equal([]string{"1 player_2 de 4"}, lines(hall[0].GetStandings()))
	s.Equal(uint32(0), hall[1].GetNumber())
	s.equalScores([]*seasonsv1.Score{{Rank: 1, CountryId: "fr", Points: 75, RoundsWon: 1}}, hall[1].GetScores())
	s.Equal([]string{"1 player_1 fr 3"}, lines(hall[1].GetStandings()))
}

func (s *testSuite) TestASeasonWithNobodyRankedHasNoPlayers() {
	s.closed(finale, rounds.Result{Country: "fr", Rank: 1, Points: 75})
	s.guest(0, 1, "fr", 3)

	s.Empty(s.onlySeason().GetStandings())
}

func (s *testSuite) TestAFailureToNameIsAnError() {
	s.closed(finale, rounds.Result{Country: "fr", Rank: 1, Points: 75})
	s.player(0, 1, "fr", 3)
	refused := errors.New("the player module is down")
	s.authors.err = refused

	_, err := hall_of_fame_query.NewPostgresQuery(s.db, s.authors).HallOfFame(s.T().Context())

	s.Require().ErrorIs(err, refused)
}

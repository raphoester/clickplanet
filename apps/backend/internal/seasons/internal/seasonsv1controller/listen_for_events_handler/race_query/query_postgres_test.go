package race_query_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/postgres_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/listen_for_events_handler/race_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db      *cppg.Postgres
	store   *postgres_round_store.Store
	seasons calendar.Calendar
	query   *race_query.PostgresQuery
}

func utc(day, hour int) time.Time {
	return time.Date(2026, 10, day, hour, 0, 0, 0, time.UTC)
}

var (
	fourteenth = rounds.Round{EndsAt: utc(14, 21)}
	fifteenth  = rounds.Round{EndsAt: utc(15, 21)}
	sixteenth  = rounds.Round{EndsAt: utc(16, 21)}
	finale     = rounds.Round{EndsAt: utc(31, 23), Finale: true}
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "seasons", migrations.FS)
	s.store = postgres_round_store.New(s.db)
	s.seasons = calendar.New(calendar.Config{List: []calendar.Entry{{Number: 0, EndsAt: utc(31, 23), Finale: 2 * time.Hour}}})
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.at(utc(16, 12))
}

func (s *testSuite) at(now time.Time) {
	s.query = race_query.NewPostgresQuery(s.db, s.seasons, cptime.NewFixedClock(now))
}

func (s *testSuite) census(round rounds.Round, held map[rounds.Country]uint32) {
	s.Require().NoError(s.store.RecordCensus(s.T().Context(), round, rounds.Census{Tiles: 100, Held: held}))
}

func (s *testSuite) closed(round rounds.Round, results ...rounds.Result) {
	s.census(round, map[rounds.Country]uint32{"fr": 1})
	s.Require().NoError(s.store.Close(s.T().Context(), round, results))
}

func (s *testSuite) race() *seasonsv1.Race {
	race, err := s.query.Race(s.T().Context())
	s.Require().NoError(err)
	return race
}

func (s *testSuite) TestTheRoundInProgressRanksEachCountryByItsAverageShareOfTheMap() {
	s.closed(fifteenth)
	s.census(sixteenth, map[rounds.Country]uint32{"fr": 30, "de": 10})
	s.census(sixteenth, map[rounds.Country]uint32{"fr": 20, "de": 10})

	race := s.race()

	s.True(proto.Equal(&seasonsv1.Round{
		Number:       2,
		EndsAtUnixMs: sixteenth.EndsAt.UnixMilli(),
		Standings: []*seasonsv1.RoundStanding{
			{Rank: 1, CountryId: "fr", Share: 0.25, Points: 25},
			{Rank: 2, CountryId: "de", Share: 0.1, Points: 18},
		},
	}, race.GetRound()), race.GetRound())
}

func (s *testSuite) TestARoundNobodyCountedYetIsShownWithNoStandings() {
	s.closed(fifteenth)

	race := s.race()

	s.True(proto.Equal(&seasonsv1.Round{Number: 2, EndsAtUnixMs: sixteenth.EndsAt.UnixMilli()}, race.GetRound()),
		race.GetRound())
}

func (s *testSuite) TestTheFinaleIsShownAsTheFinale() {
	s.at(utc(31, 22))
	s.census(finale, map[rounds.Country]uint32{"fr": 50})

	race := s.race()

	s.True(proto.Equal(&seasonsv1.Round{
		Number:       1,
		EndsAtUnixMs: finale.EndsAt.UnixMilli(),
		Finale:       true,
		Standings:    []*seasonsv1.RoundStanding{{Rank: 1, CountryId: "fr", Share: 0.5, Points: 75}},
	}, race.GetRound()), race.GetRound())
}

func (s *testSuite) TestTheScoresAreThePointsOfTheClosedRoundsBestFirst() {
	s.closed(fourteenth,
		rounds.Result{Country: "de", Rank: 1, Points: 25},
		rounds.Result{Country: "fr", Rank: 2, Points: 18},
		rounds.Result{Country: "es", Rank: 11, Points: 0})
	s.closed(fifteenth,
		rounds.Result{Country: "fr", Rank: 1, Points: 25},
		rounds.Result{Country: "de", Rank: 2, Points: 18})
	s.census(sixteenth, map[rounds.Country]uint32{"it": 99})

	race := s.race()

	want := []*seasonsv1.Score{
		{Rank: 1, CountryId: "de", Points: 43, RoundsWon: 1},
		{Rank: 1, CountryId: "fr", Points: 43, RoundsWon: 1},
	}
	s.Require().Len(race.GetScores(), len(want), race.GetScores())
	for i := range want {
		s.True(proto.Equal(want[i], race.GetScores()[i]), race.GetScores()[i])
	}
}

func (s *testSuite) TestThereIsNoRaceOutsideASeason() {
	s.at(utc(31, 23))
	s.closed(fifteenth, rounds.Result{Country: "fr", Rank: 1, Points: 25})

	s.True(proto.Equal(&seasonsv1.Race{}, s.race()))
}

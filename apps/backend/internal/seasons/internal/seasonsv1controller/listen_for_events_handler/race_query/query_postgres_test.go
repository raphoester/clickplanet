package race_query_test

import (
	"context"
	"errors"
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

type stubTerritory struct {
	held  map[rounds.Country]uint32
	err   error
	asked int
}

func (s *stubTerritory) Snapshot(context.Context) (rounds.Snapshot, error) {
	s.asked++
	return rounds.Snapshot{Tiles: 100, Held: s.held}, s.err
}

type testSuite struct {
	suite.Suite

	db        *cppg.Postgres
	store     *postgres_round_store.Store
	territory *stubTerritory
	seasons   calendar.Calendar
	query     *race_query.PostgresQuery
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
	s.territory = &stubTerritory{}
	s.at(utc(16, 12))
}

func (s *testSuite) at(now time.Time) {
	s.query = race_query.NewPostgresQuery(s.db, s.territory, s.seasons, cptime.NewFixedClock(now))
}

func (s *testSuite) open(round rounds.Round) {
	s.Require().NoError(s.store.Open(s.T().Context(), round))
}

func (s *testSuite) closed(round rounds.Round, results ...rounds.Result) {
	s.open(round)
	_, err := s.store.Close(s.T().Context(), round, results)
	s.Require().NoError(err)
}

func (s *testSuite) race() *seasonsv1.Race {
	race, err := s.query.Race(s.T().Context())
	s.Require().NoError(err)
	return race
}

func (s *testSuite) TestTheRoundInProgressRanksEachCountryByItsShareOfTheMapNow() {
	s.closed(fifteenth)
	s.open(sixteenth)
	s.territory.held = map[rounds.Country]uint32{"fr": 25, "de": 10}

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

func (s *testSuite) TestARoundNobodyHoldsGroundInIsShownWithNoStandings() {
	s.closed(fifteenth)

	race := s.race()

	s.True(proto.Equal(&seasonsv1.Round{Number: 2, EndsAtUnixMs: sixteenth.EndsAt.UnixMilli()}, race.GetRound()),
		race.GetRound())
}

func (s *testSuite) TestTheFinaleIsShownAsTheFinale() {
	s.at(utc(31, 22))
	s.territory.held = map[rounds.Country]uint32{"fr": 50}

	race := s.race()

	s.True(proto.Equal(&seasonsv1.Round{
		Number:       1,
		EndsAtUnixMs: finale.EndsAt.UnixMilli(),
		Finale:       true,
		Standings:    []*seasonsv1.RoundStanding{{Rank: 1, CountryId: "fr", Share: 0.5, Points: 75}},
	}, race.GetRound()), race.GetRound())
}

func (s *testSuite) TestAMapThatCannotBeReadIsAnError() {
	unreachable := errors.New("planet is down")
	s.territory.err = unreachable

	_, err := s.query.Race(s.T().Context())

	s.ErrorIs(err, unreachable)
}

func (s *testSuite) TestTheScoresAreThePointsOfTheClosedRoundsBestFirst() {
	s.closed(fourteenth,
		rounds.Result{Country: "de", Rank: 1, Points: 25},
		rounds.Result{Country: "fr", Rank: 2, Points: 18},
		rounds.Result{Country: "es", Rank: 11, Points: 0})
	s.closed(fifteenth,
		rounds.Result{Country: "fr", Rank: 1, Points: 25},
		rounds.Result{Country: "de", Rank: 2, Points: 18})
	s.territory.held = map[rounds.Country]uint32{"it": 99}

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

func (s *testSuite) TestOutsideASeasonTheRaceIsOnlyTheRoundClosedLast() {
	s.at(utc(31, 23))
	s.closed(fifteenth, rounds.Result{Country: "fr", Rank: 1, Points: 25})

	race := s.race()

	s.Nil(race.GetRound())
	s.Empty(race.GetScores())
	s.Equal(uint32(1), race.GetClosed().GetNumber())
	s.Zero(s.territory.asked, "no round is in progress to read the map for")
}

func (s *testSuite) TestBeforeAnyRoundClosesTheRaceCarriesNoClosedRoundAndNoScores() {
	s.open(sixteenth)
	s.territory.held = map[rounds.Country]uint32{"fr": 30}

	race := s.race()

	s.Nil(race.GetClosed())
	s.Empty(race.GetScores())
}

func (s *testSuite) TestTheRoundClosedLastCarriesItsResultsWithTheShareHeldAtItsEndAndTheTableBeforeAndAfterIt() {
	s.closed(fourteenth,
		rounds.Result{Country: "de", Rank: 1, Points: 25},
		rounds.Result{Country: "fr", Rank: 2, Points: 18})
	s.closed(fifteenth,
		rounds.Result{Country: "fr", Rank: 1, Points: 25, Share: 0.25},
		rounds.Result{Country: "de", Rank: 2, Points: 18, Share: 0.1},
		rounds.Result{Country: "es", Rank: 11, Points: 0, Share: 0.01})
	s.open(sixteenth)
	s.territory.held = map[rounds.Country]uint32{"it": 99}

	s.True(proto.Equal(&seasonsv1.ClosedRound{
		Season:        0,
		Number:        2,
		EndedAtUnixMs: fifteenth.EndsAt.UnixMilli(),
		Standings: []*seasonsv1.RoundStanding{
			{Rank: 1, CountryId: "fr", Share: 0.25, Points: 25},
			{Rank: 2, CountryId: "de", Share: 0.1, Points: 18},
		},
		Before: []*seasonsv1.Score{
			{Rank: 1, CountryId: "de", Points: 25, RoundsWon: 1},
			{Rank: 2, CountryId: "fr", Points: 18},
		},
		After: []*seasonsv1.Score{
			{Rank: 1, CountryId: "de", Points: 43, RoundsWon: 1},
			{Rank: 1, CountryId: "fr", Points: 43, RoundsWon: 1},
		},
	}, s.race().GetClosed()), s.race().GetClosed())
}

func (s *testSuite) TestTheFirstRoundClosedHasNothingBeforeIt() {
	s.closed(fifteenth, rounds.Result{Country: "fr", Rank: 1, Points: 25})

	closed := s.race().GetClosed()

	s.Empty(closed.GetBefore())
	s.Len(closed.GetAfter(), 1)
}

func (s *testSuite) TestTheFinaleClosedIsShownAsTheFinale() {
	s.at(utc(31, 23))
	s.closed(finale, rounds.Result{Country: "fr", Rank: 1, Points: 75})

	closed := s.race().GetClosed()

	s.True(closed.GetFinale())
	s.Equal(finale.EndsAt.UnixMilli(), closed.GetEndedAtUnixMs())
}

func (s *testSuite) TestTheTableAfterTheFinaleIsTheTableTheSeasonEndsWith() {
	s.closed(fourteenth, rounds.Result{Country: "fr", Rank: 1, Points: 25}, rounds.Result{Country: "de", Rank: 2, Points: 18})
	s.closed(fifteenth, rounds.Result{Country: "de", Rank: 1, Points: 25}, rounds.Result{Country: "fr", Rank: 2, Points: 18})
	s.closed(sixteenth, rounds.Result{Country: "it", Rank: 1, Points: 25}, rounds.Result{Country: "nz", Rank: 11})
	earlier, err := s.store.ClosedIn(s.T().Context(), 0)
	s.Require().NoError(err)
	results := []rounds.Result{{Country: "de", Rank: 1, Points: 75}, {Country: "it", Rank: 2, Points: 54}, {Country: "fr", Rank: 2, Points: 54}}
	s.closed(finale, results...)
	s.at(utc(31, 23).Add(time.Minute))

	end, over := rounds.Closed{Round: finale, Number: 4, Results: results}.SeasonEnd(earlier)

	s.Require().True(over)
	ended := make([]*seasonsv1.Score, 0, len(end.Table))
	for _, placed := range end.Table {
		ended = append(ended, &seasonsv1.Score{
			Rank: placed.Rank, CountryId: string(placed.Country), Points: uint32(placed.Points), RoundsWon: placed.RoundsWon,
		})
	}
	after := s.race().GetClosed().GetAfter()
	s.Len(after, len(ended))
	for i := range ended {
		s.True(proto.Equal(ended[i], after[i]), "the race shows %v, the season ends with %v", after[i], ended[i])
	}
}

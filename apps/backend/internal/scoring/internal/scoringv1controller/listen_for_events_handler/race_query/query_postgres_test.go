package race_query_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/postgres_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/listen_for_events_handler/race_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type stubTerritory struct {
	held map[rounds.Country]uint32
}

func (s *stubTerritory) Snapshot(context.Context) (rounds.Snapshot, error) {
	return rounds.Snapshot{Tiles: 100, Held: s.held}, nil
}

type stubAuthors struct {
	authors map[rounds.AccountID]*playerv1.Author
}

func (s stubAuthors) Authors(_ context.Context, accounts []rounds.AccountID) (map[rounds.AccountID]*playerv1.Author, error) {
	found := map[rounds.AccountID]*playerv1.Author{}
	for _, account := range accounts {
		if author, known := s.authors[account]; known {
			found[account] = author
		}
	}
	return found, nil
}

type testSuite struct {
	suite.Suite

	db        *cppg.Postgres
	store     *postgres_round_store.Store
	territory *stubTerritory
	authors   stubAuthors
	seasons   calendar.Calendar
	query     *race_query.PostgresQuery
}

func utc(day, hour int) time.Time {
	return time.Date(2026, 10, day, hour, 0, 0, 0, time.UTC)
}

var (
	fourteenth = rounds.Round{EndsAt: utc(14, 21), Times: 1}
	fifteenth  = rounds.Round{EndsAt: utc(15, 21), Times: 1}
	sixteenth  = rounds.Round{EndsAt: utc(16, 21), Times: 1}
	finale     = rounds.Round{EndsAt: utc(31, 23), Finale: true, Own: true, StartsAt: utc(31, 21), Times: 3}
	// battle is a round of its own on the sixteenth, before the day ends.
	battle = rounds.Round{EndsAt: utc(16, 20), Own: true, StartsAt: utc(16, 18), Times: 2}
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "scoring", migrations.FS)
	s.store = postgres_round_store.New(s.db)
	s.seasons = calendar.New(calendar.Config{DailyCutoff: 21 * time.Hour, List: []calendar.Entry{{Number: 0, EndsAt: utc(31, 23)}}})
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.territory = &stubTerritory{}
	s.authors = stubAuthors{authors: map[rounds.AccountID]*playerv1.Author{}}
	s.at(utc(16, 12))
}

func (s *testSuite) at(now time.Time) {
	s.query = race_query.NewPostgresQuery(s.db, s.territory, s.authors, s.seasons, cptime.NewFixedClock(now))
}

func (s *testSuite) open(round rounds.Round) {
	s.Require().NoError(s.store.Open(s.T().Context(), round))
}

func (s *testSuite) closed(round rounds.Round, results ...rounds.Result) {
	s.open(round)
	_, err := s.store.Close(s.T().Context(), round, results)
	s.Require().NoError(err)
}

func (s *testSuite) race() *scoringv1.Race {
	race, err := s.query.Race(s.T().Context())
	s.Require().NoError(err)
	return race
}

func (s *testSuite) TestTheRoundInProgressRanksEachCountryByItsShareOfTheMapNow() {
	s.closed(fifteenth)
	s.open(sixteenth)
	s.territory.held = map[rounds.Country]uint32{"fr": 25, "de": 10}

	race := s.race()

	s.True(proto.Equal(&scoringv1.Round{
		Number:       2,
		EndsAtUnixMs: sixteenth.EndsAt.UnixMilli(),
		Standings: []*scoringv1.RoundStanding{
			{Rank: 1, CountryId: "fr", Share: 0.25, Points: 25},
			{Rank: 2, CountryId: "de", Share: 0.1, Points: 18},
		},
		Times: 1,
	}, race.GetRound()), race.GetRound())
}

func (s *testSuite) TestARoundNobodyHoldsGroundInIsShownWithNoStandings() {
	s.closed(fifteenth)

	race := s.race()

	s.True(proto.Equal(&scoringv1.Round{Number: 2, EndsAtUnixMs: sixteenth.EndsAt.UnixMilli(), Times: 1}, race.GetRound()),
		race.GetRound())
}

func (s *testSuite) TestTheFinaleIsTheRoundOfItsOwnThatRunsToTheSeasonsEnd() {
	s.open(finale)
	s.at(utc(31, 22))
	s.territory.held = map[rounds.Country]uint32{"fr": 50}

	race := s.race()

	s.True(proto.Equal(&scoringv1.Round{
		Number:       1,
		EndsAtUnixMs: finale.EndsAt.UnixMilli(),
		Finale:       true,
		Standings:    []*scoringv1.RoundStanding{{Rank: 1, CountryId: "fr", Share: 0.5, Points: 75}},
		Own:          true,
		Times:        3,
	}, race.GetRound()), race.GetRound())
}

func (s *testSuite) TestFromTheLastCutoffToTheSeasonsEndWithNoRoundOfItsOwnNoRoundRuns() {
	s.at(utc(31, 22))

	s.Nil(s.race().GetRound())
}

func (s *testSuite) TestARoundOfItsOwnThatEndsBeforeTheDayIsTheRoundInProgress() {
	s.closed(fifteenth)
	s.open(sixteenth)
	s.open(battle)
	s.at(utc(16, 19))
	s.territory.held = map[rounds.Country]uint32{"fr": 25}

	race := s.race()

	s.True(proto.Equal(&scoringv1.Round{
		Number:       1,
		EndsAtUnixMs: battle.EndsAt.UnixMilli(),
		Standings:    []*scoringv1.RoundStanding{{Rank: 1, CountryId: "fr", Share: 0.25, Points: 50}},
		Own:          true,
		Times:        2,
	}, race.GetRound()), race.GetRound())

	s.at(utc(16, 20))
	s.Equal(uint32(2), s.race().GetRound().GetNumber(), "the day is the second, the round of its own counted apart")
	s.False(s.race().GetRound().GetOwn())
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

	want := []*scoringv1.Score{
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

	s.True(proto.Equal(&scoringv1.ClosedRound{
		Season:        0,
		Number:        2,
		EndedAtUnixMs: fifteenth.EndsAt.UnixMilli(),
		Standings: []*scoringv1.RoundStanding{
			{Rank: 1, CountryId: "fr", Share: 0.25, Points: 25},
			{Rank: 2, CountryId: "de", Share: 0.1, Points: 18},
		},
		Before: []*scoringv1.Score{
			{Rank: 1, CountryId: "de", Points: 25, RoundsWon: 1},
			{Rank: 2, CountryId: "fr", Points: 18},
		},
		After: []*scoringv1.Score{
			{Rank: 1, CountryId: "de", Points: 43, RoundsWon: 1},
			{Rank: 1, CountryId: "fr", Points: 43, RoundsWon: 1},
		},
		Times: 1,
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

	season, _ := s.seasons.Current(utc(31, 22))
	end := rounds.SeasonEndOf(season, append(earlier, rounds.Closed{Round: finale, Number: 1, Results: results}))

	ended := make([]*scoringv1.Score, 0, len(end.Table))
	for _, placed := range end.Table {
		ended = append(ended, &scoringv1.Score{
			Rank: placed.Rank, CountryId: string(placed.Country), Points: uint32(placed.Points), RoundsWon: placed.RoundsWon,
		})
	}
	after := s.race().GetClosed().GetAfter()
	s.Len(after, len(ended))
	for i := range ended {
		s.True(proto.Equal(ended[i], after[i]), "the race shows %v, the season ends with %v", after[i], ended[i])
	}
}

var (
	ana   = rounds.AccountID{1}
	ben   = rounds.AccountID{2}
	guest = rounds.AccountID{3}
	cleo  = rounds.AccountID{4}
)

func (s *testSuite) take(account rounds.AccountID, country rounds.Country, position uint64) {
	s.Require().NoError(s.store.Count(s.T().Context(), rounds.Take{
		Account: account, Country: country, At: finale.StartsAt.Add(time.Minute), Position: position,
	}))
}

func (s *testSuite) TestARoundOfItsOwnClosedCarriesItsStartTheTilesEachFlagTookAndItsNamedPlayers() {
	s.open(finale)
	position := uint64(0)
	for account, takes := range map[rounds.AccountID]map[rounds.Country]int{
		ana:   {"fr": 3, "be": 1},
		ben:   {"de": 2},
		guest: {"de": 9},
		cleo:  {"it": 2},
	} {
		for country, count := range takes {
			for range count {
				position++
				s.take(account, country, position)
			}
		}
	}
	s.authors.authors[ana] = &playerv1.Author{AccountId: ana.String(), Name: "ana"}
	s.authors.authors[ben] = &playerv1.Author{AccountId: ben.String(), Name: "ben"}
	s.authors.authors[guest] = &playerv1.Author{AccountId: guest.String(), Guest: true}
	s.authors.authors[cleo] = &playerv1.Author{AccountId: cleo.String(), Name: "cleo"}
	_, err := s.store.Close(s.T().Context(), finale, []rounds.Result{{Country: "de", Rank: 1, Points: 75}})
	s.Require().NoError(err)
	s.at(utc(31, 23))

	closed := s.race().GetClosed()

	s.True(closed.GetOwn())
	s.True(closed.GetFinale())
	s.Equal(uint32(3), closed.GetTimes())
	s.Equal(finale.StartsAt.UnixMilli(), closed.GetStartedAtUnixMs())
	s.True(proto.Equal(&scoringv1.ClosedRound{Takes: []*scoringv1.RoundTakes{
		{CountryId: "de", Tiles: 11}, {CountryId: "fr", Tiles: 3}, {CountryId: "it", Tiles: 2}, {CountryId: "be", Tiles: 1},
	}}, &scoringv1.ClosedRound{Takes: closed.GetTakes()}), "%v", closed.GetTakes())
	want := []*scoringv1.Standing{
		{Rank: 1, AccountId: ana.String(), Name: "ana", CountryId: "fr", Tiles: 3},
		{Rank: 2, AccountId: ben.String(), Name: "ben", CountryId: "de", Tiles: 2},
		{Rank: 2, AccountId: cleo.String(), Name: "cleo", CountryId: "it", Tiles: 2},
	}
	s.Require().Len(closed.GetPlayers(), len(want), closed.GetPlayers())
	for i := range want {
		s.True(proto.Equal(want[i], closed.GetPlayers()[i]), "%v", closed.GetPlayers()[i])
	}
}

func (s *testSuite) TestADayClosedCarriesNoNumbersOfARoundOfItsOwn() {
	s.closed(fifteenth, rounds.Result{Country: "fr", Rank: 1, Points: 25})

	closed := s.race().GetClosed()

	s.False(closed.GetOwn())
	s.Zero(closed.GetStartedAtUnixMs())
	s.Empty(closed.GetTakes())
	s.Empty(closed.GetPlayers())
}

func (s *testSuite) TestTheScoresCountTheRoundsOfTheirOwn() {
	s.closed(fifteenth, rounds.Result{Country: "fr", Rank: 1, Points: 25})
	s.closed(battle, rounds.Result{Country: "de", Rank: 1, Points: 50})
	s.at(utc(16, 20))

	scores := s.race().GetScores()

	s.Require().Len(scores, 2)
	s.Equal("de", scores[0].GetCountryId())
	s.Equal(uint32(50), scores[0].GetPoints())
}

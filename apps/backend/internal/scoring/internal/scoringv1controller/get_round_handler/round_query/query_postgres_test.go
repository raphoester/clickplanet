package round_query_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/postgres_book_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/postgres_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/departures"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/departures/postgres_departure_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/players/postgres_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_round_handler/round_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_scores_handler/scores_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type stubTerritory struct {
	held map[books.Country]uint32
}

func (s *stubTerritory) Snapshot(context.Context) (books.Snapshot, error) {
	return books.Snapshot{Tiles: 100, Held: s.held}, nil
}

func utc(day, hour int) time.Time {
	return time.Date(2026, 10, day, hour, 0, 0, 0, time.UTC)
}

var (
	seasonZero = books.NewBook("season-0", utc(8, 0))
	day23      = books.Round{ID: "season-0/day-23", Book: seasonZero.ID, Times: 1, Ranking: books.GainRanking{}, StartsAt: utc(29, 21)}
	day24      = books.Round{ID: "season-0/day-24", Book: seasonZero.ID, Times: 1, Ranking: books.GainRanking{}, StartsAt: utc(30, 21)}
	battle     = books.Round{ID: "0192f5a0-7c00-7000-8000-000000000031", Book: seasonZero.ID, Times: 3, Ranking: books.GainRanking{}, StartsAt: utc(31, 21)}
)

type testSuite struct {
	suite.Suite

	db        *cppg.Postgres
	rounds    *postgres_round_store.Store
	territory *stubTerritory
	query     *round_query.PostgresQuery
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "scoring", migrations.FS)
	s.rounds = postgres_round_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.territory = &stubTerritory{held: map[books.Country]uint32{}}
	s.query = round_query.NewPostgresQuery(s.db, s.territory, scores_query.NewPostgresQuery(s.db))
	_, err := postgres_book_store.New(s.db).Open(s.T().Context(), seasonZero)
	s.Require().NoError(err)
	for _, round := range []books.Round{day23, day24, battle} {
		_, err := s.rounds.Open(s.T().Context(), round, books.Start{})
		s.Require().NoError(err)
	}
}

func (s *testSuite) score(round books.Round, endsAt time.Time, held map[books.Country]uint32) {
	_, err := s.rounds.Score(s.T().Context(), round.ScoredAt(endsAt, books.Start{}, books.Snapshot{Tiles: 100, Held: held}))
	s.Require().NoError(err)
}

func (s *testSuite) round(id books.RoundID) *scoringv1.Round {
	round, err := s.query.Round(s.T().Context(), id)
	s.Require().NoError(err)
	return round
}

func (s *testSuite) TestARoundNeverOpenedIsNone() {
	_, err := s.query.Round(s.T().Context(), "season-0/day-25")
	s.Require().ErrorIs(err, round_query.ErrNoRound)

	_, found, err := s.query.Live(s.T().Context(), "season-0/day-25")
	s.Require().NoError(err)
	s.False(found, "a stream shows it once it opens")
}

func (s *testSuite) TestAnOpenRoundRanksTheMapNowTimesItsTimes() {
	s.territory.held = map[books.Country]uint32{"fr": 30, "de": 20}

	s.True(proto.Equal(&scoringv1.Round{
		RoundId: string(battle.ID), BookId: "season-0", Times: 3, StartedAtUnixMs: utc(31, 21).UnixMilli(),
		Standings: []*scoringv1.RoundStanding{
			{Rank: 1, CountryId: "fr", Share: 0.3, Points: 75}, {Rank: 2, CountryId: "de", Share: 0.2, Points: 54},
		},
		Takes: []*scoringv1.RoundTakes{}, Players: []*scoringv1.Standing{},
	}, s.round(battle.ID)), s.round(battle.ID))
}

func (s *testSuite) TestAnOpenRoundRanksTheTilesGainedSinceItOpened() {
	day25 := books.Round{ID: "season-0/day-25", Book: seasonZero.ID, Times: 1, Ranking: books.GainRanking{}, StartsAt: utc(31, 23)}
	_, err := s.rounds.Open(s.T().Context(), day25, books.Start{"fr": 30, "de": 5})
	s.Require().NoError(err)
	s.territory.held = map[books.Country]uint32{"fr": 25, "de": 20}

	s.True(proto.Equal(&scoringv1.Round{
		RoundId: string(day25.ID), BookId: "season-0", Times: 1, StartedAtUnixMs: utc(31, 23).UnixMilli(),
		Standings: []*scoringv1.RoundStanding{
			{Rank: 1, CountryId: "de", Share: 0.2, Points: 25}, {Rank: 2, CountryId: "fr", Share: 0.25},
		},
		Takes: []*scoringv1.RoundTakes{}, Players: []*scoringv1.Standing{},
		Start: []*scoringv1.RoundStart{{CountryId: "fr", Tiles: 30}, {CountryId: "de", Tiles: 5}},
	}, s.round(day25.ID)), s.round(day25.ID))
}

func (s *testSuite) TestAnOpenDayRankedOnTheGroundHeldAndGainedRanksBoth() {
	day25 := books.Round{ID: "season-0/day-25", Book: seasonZero.ID, Times: 1, Ranking: books.HeldAndGainRanking{}, StartsAt: utc(31, 23)}
	_, err := s.rounds.Open(s.T().Context(), day25, books.Start{"fr": 30, "de": 5})
	s.Require().NoError(err)
	s.territory.held = map[books.Country]uint32{"fr": 30, "de": 20}

	s.True(proto.Equal(&scoringv1.Round{
		RoundId: string(day25.ID), BookId: "season-0", Times: 1, StartedAtUnixMs: utc(31, 23).UnixMilli(),
		Standings: []*scoringv1.RoundStanding{
			{Rank: 1, CountryId: "de", Share: 0.2, Points: 25}, {Rank: 2, CountryId: "fr", Share: 0.3, Points: 18},
		},
		Takes: []*scoringv1.RoundTakes{}, Players: []*scoringv1.Standing{},
		Start: []*scoringv1.RoundStart{{CountryId: "fr", Tiles: 30}, {CountryId: "de", Tiles: 5}},
	}, s.round(day25.ID)), s.round(day25.ID))
}

func (s *testSuite) TestAScoredRoundShowsItsResultsAndTheBooksScoresBeforeAndAfterIt() {
	s.score(day23, utc(30, 21), map[books.Country]uint32{"fr": 30, "de": 20})
	s.score(day24, utc(31, 21), map[books.Country]uint32{"de": 30, "fr": 20, "it": 0})
	s.territory.held = map[books.Country]uint32{"es": 99}

	s.True(proto.Equal(&scoringv1.Round{
		RoundId: string(day24.ID), BookId: "season-0", Times: 1,
		StartedAtUnixMs: utc(30, 21).UnixMilli(), EndedAtUnixMs: utc(31, 21).UnixMilli(),
		Standings: []*scoringv1.RoundStanding{
			{Rank: 1, CountryId: "de", Share: 0.3, Points: 25}, {Rank: 2, CountryId: "fr", Share: 0.2, Points: 18},
		},
		Takes: []*scoringv1.RoundTakes{}, Players: []*scoringv1.Standing{},
		Before: []*scoringv1.Score{
			{Rank: 1, CountryId: "fr", Points: 25, RoundsWon: 1}, {Rank: 2, CountryId: "de", Points: 18},
		},
		After: []*scoringv1.Score{
			{Rank: 1, CountryId: "de", Points: 43, RoundsWon: 1}, {Rank: 1, CountryId: "fr", Points: 43, RoundsWon: 1},
		},
	}, s.round(day24.ID)), s.round(day24.ID))
}

func (s *testSuite) TestARoundNamesTheFlagsAndTheSignedInPlayersThatTookTheMost() {
	ada, bob, guest, leaving := books.AccountID{15: 1}, books.AccountID{15: 2}, books.AccountID{15: 3}, books.AccountID{15: 4}
	for _, player := range []books.AccountID{ada, bob, leaving} {
		s.Require().NoError(postgres_player_store.New(s.db).Keep(s.T().Context(), player))
	}
	s.Require().NoError(postgres_departure_store.New(s.db).Change(s.T().Context(), leaving,
		func(departure departures.Departure) departures.Departure {
			return departure.Decided(departures.Leaves(utc(31, 22)))
		}))
	position := uint64(0)
	take := func(account books.AccountID, country books.Country, tiles int) {
		takes := make([]books.Take, 0, tiles)
		for range tiles {
			position++
			takes = append(takes, books.Take{Account: account, Country: country, At: battle.StartsAt, Position: position})
		}
		s.Require().NoError(s.rounds.Count(s.T().Context(), map[books.RoundID][]books.Take{battle.ID: takes}))
	}
	take(ada, "fr", 3)
	take(ada, "be", 1)
	take(bob, "de", 2)
	take(guest, "de", 9)
	take(leaving, "it", 7)

	round := s.round(battle.ID)

	s.True(proto.Equal(&scoringv1.Round{
		RoundId: string(battle.ID), BookId: "season-0", Times: 3, StartedAtUnixMs: utc(31, 21).UnixMilli(),
		Standings: []*scoringv1.RoundStanding{},
		Takes: []*scoringv1.RoundTakes{
			{CountryId: "de", Tiles: 11}, {CountryId: "it", Tiles: 7}, {CountryId: "fr", Tiles: 3}, {CountryId: "be", Tiles: 1},
		},
		Players: []*scoringv1.Standing{
			{Rank: 1, AccountId: ada.String(), CountryId: "fr", Tiles: 3},
			{Rank: 2, AccountId: bob.String(), CountryId: "de", Tiles: 2},
		},
	}, round), round)
}

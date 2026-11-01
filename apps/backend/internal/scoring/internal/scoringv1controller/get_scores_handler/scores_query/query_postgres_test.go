package scores_query_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/postgres_book_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/postgres_grant_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/postgres_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_scores_handler/scores_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

func utc(day, hour int) time.Time {
	return time.Date(2026, 10, day, hour, 0, 0, 0, time.UTC)
}

var (
	seasonZero = books.NewBook("season-0", utc(8, 0))
	day24      = books.Round{ID: "season-0/day-24", Book: seasonZero.ID, Times: 1, StartsAt: utc(30, 21)}
	battle     = books.Round{ID: "0192f5a0-7c00-7000-8000-000000000031", Book: seasonZero.ID, Times: 3, StartsAt: utc(31, 21)}
)

type testSuite struct {
	suite.Suite

	db    *cppg.Postgres
	books *postgres_book_store.Store
	query *scores_query.PostgresQuery
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "scoring", migrations.FS)
	s.books = postgres_book_store.New(s.db)
	s.query = scores_query.NewPostgresQuery(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	_, err := s.books.Open(s.T().Context(), seasonZero)
	s.Require().NoError(err)
	rounds := postgres_round_store.New(s.db)
	for _, scored := range []books.Scored{
		day24.ScoredAt(utc(31, 21), books.Snapshot{Tiles: 4, Held: map[books.Country]uint32{"fr": 3, "de": 1}}),
		battle.ScoredAt(utc(31, 23), books.Snapshot{Tiles: 5, Held: map[books.Country]uint32{"de": 3, "fr": 2}}),
	} {
		_, err := rounds.Open(s.T().Context(), scored.Round)
		s.Require().NoError(err)
		_, err = rounds.Score(s.T().Context(), scored)
		s.Require().NoError(err)
	}
}

func (s *testSuite) TestTheScoresAreTheBooksCountriesByPoints() {
	scores, err := s.query.Scores(s.T().Context(), seasonZero.ID)

	s.Require().NoError(err)
	s.True(proto.Equal(&scoringv1.Scores{BookId: "season-0", Scores: []*scoringv1.Score{
		{Rank: 1, CountryId: "de", Points: 93, RoundsWon: 1}, {Rank: 2, CountryId: "fr", Points: 79, RoundsWon: 1},
	}}, scores), scores)
}

func (s *testSuite) TestABookWithNoRoundScoredHasNoScores() {
	scores, err := s.query.Scores(s.T().Context(), "season-1")

	s.Require().NoError(err)
	s.True(proto.Equal(&scoringv1.Scores{BookId: "season-1", Scores: []*scoringv1.Score{}}, scores), scores)
}

func (s *testSuite) TestTheScoresAroundARoundAreBeforeItAndThroughIt() {
	before, after, err := s.query.Around(s.T().Context(), seasonZero.ID, utc(31, 21))

	s.Require().NoError(err)
	s.Empty(before)
	s.Len(after, 2)
	s.True(proto.Equal(&scoringv1.Score{Rank: 1, CountryId: "fr", Points: 25, RoundsWon: 1}, after[0]), after[0])
}

func (s *testSuite) TestTheHistoryIsEachRoundsResultsAndTheTableAfterIt() {
	history, err := s.query.History(s.T().Context(), seasonZero.ID)

	s.Require().NoError(err)
	s.Require().Len(history, 2)
	s.True(proto.Equal(&scoringv1.BookStep{
		RoundId: "season-0/day-24", EndedAtUnixMs: utc(31, 21).UnixMilli(),
		Results: []*scoringv1.RoundStanding{
			{Rank: 1, CountryId: "fr", Share: 0.75, Points: 25}, {Rank: 2, CountryId: "de", Share: 0.25, Points: 18},
		},
		Table: []*scoringv1.Score{{Rank: 1, CountryId: "fr", Points: 25, RoundsWon: 1}, {Rank: 2, CountryId: "de", Points: 18}},
	}, history[0]), "%v", history[0])
	s.True(proto.Equal(&scoringv1.Score{Rank: 1, CountryId: "de", Points: 93, RoundsWon: 1}, history[1].GetTable()[0]), "%v", history[1])
}

func (s *testSuite) TestTheScoresCountTheBooksGrantsAndAroundARoundTheOnesBeforeItsEnd() {
	grants := postgres_grant_store.New(s.db)
	for _, grant := range []books.Grant{
		{ID: "before the book", Country: "fr", Points: 3, At: utc(7, 0)},
		{ID: "between the rounds", Country: "de", Points: 2, At: utc(31, 22)},
	} {
		_, err := grants.Keep(s.T().Context(), grant)
		s.Require().NoError(err)
	}

	scores, err := s.query.Scores(s.T().Context(), seasonZero.ID)
	s.Require().NoError(err)
	before, after, err := s.query.Around(s.T().Context(), seasonZero.ID, utc(31, 23))
	s.Require().NoError(err)

	s.True(proto.Equal(&scoringv1.Scores{BookId: "season-0", Scores: []*scoringv1.Score{
		{Rank: 1, CountryId: "de", Points: 95, RoundsWon: 1}, {Rank: 2, CountryId: "fr", Points: 79, RoundsWon: 1},
	}}, scores), scores)
	s.True(proto.Equal(&scoringv1.Score{Rank: 2, CountryId: "de", Points: 20}, before[1]), before[1])
	s.True(proto.Equal(&scoringv1.Score{Rank: 1, CountryId: "de", Points: 95, RoundsWon: 1}, after[0]), after[0])
}

func (s *testSuite) TestTheTableOfAClosedBookCarriesItsTiebreak() {
	_, err := s.books.Close(s.T().Context(), seasonZero.ClosedAt(utc(31, 23), battle.ID))
	s.Require().NoError(err)

	table, err := s.query.Table(s.T().Context(), seasonZero.ID)

	s.Require().NoError(err)
	s.True(proto.Equal(&scoringv1.GetTableResponse{Closed: true, Table: []*scoringv1.TablePlace{
		{CountryId: "de", Rank: 1, Points: 93, RoundsWon: 1, TiebreakPoints: 75},
		{CountryId: "fr", Rank: 2, Points: 79, RoundsWon: 1, TiebreakPoints: 54},
	}}, table), table)
}

func (s *testSuite) TestTheTableOfAnOpenBookIsNotClosedAndHasNoTiebreak() {
	table, err := s.query.Table(s.T().Context(), seasonZero.ID)

	s.Require().NoError(err)
	s.False(table.GetClosed())
	s.Zero(table.GetTable()[0].GetTiebreakPoints())
}

func (s *testSuite) TestTheTableOfABookNeverOpenedIsNone() {
	_, err := s.query.Table(s.T().Context(), "season-1")

	s.Require().ErrorIs(err, scores_query.ErrNoBook)
}

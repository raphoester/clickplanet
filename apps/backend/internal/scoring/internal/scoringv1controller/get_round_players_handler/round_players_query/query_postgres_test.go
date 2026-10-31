package round_players_query_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/postgres_book_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/postgres_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_round_players_handler/round_players_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

var (
	seasonZero = books.NewBook("season-0", time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC))
	finale     = books.Round{
		ID: "season-0/finale", Book: seasonZero.ID, Times: 3, StartsAt: time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC),
	}
)

type testSuite struct {
	suite.Suite

	db       *cppg.Postgres
	rounds   *postgres_round_store.Store
	query    *round_players_query.PostgresQuery
	position uint64
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "scoring", migrations.FS)
	s.rounds = postgres_round_store.New(s.db)
	s.query = round_players_query.NewPostgresQuery(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	_, err := postgres_book_store.New(s.db).Open(s.T().Context(), seasonZero)
	s.Require().NoError(err)
	_, err = s.rounds.Open(s.T().Context(), finale)
	s.Require().NoError(err)
}

func (s *testSuite) take(account books.AccountID, country books.Country, tiles int) {
	for range tiles {
		s.position++
		s.Require().NoError(s.rounds.Count(s.T().Context(), []books.RoundID{finale.ID},
			books.Take{Account: account, Country: country, At: finale.StartsAt, Position: s.position}))
	}
}

func (s *testSuite) TestEveryAccountOfTheRoundIsAnsweredUnderTheFlagItTookTheMostFor() {
	ada, guest := books.AccountID{15: 1}, books.AccountID{15: 2}
	s.take(ada, "fr", 3)
	s.take(ada, "be", 1)
	s.take(guest, "de", 2)

	players, err := s.query.Players(s.T().Context(), finale.ID)

	s.Require().NoError(err)
	s.Require().Len(players, 2)
	s.True(proto.Equal(&scoringv1.RoundPlayer{AccountId: ada.String(), CountryId: "fr", Tiles: 3}, players[0]), players[0])
	s.True(proto.Equal(&scoringv1.RoundPlayer{AccountId: guest.String(), CountryId: "de", Tiles: 2}, players[1]), players[1])
}

func (s *testSuite) TestARoundNobodyTookTilesInHasNoPlayers() {
	players, err := s.query.Players(s.T().Context(), finale.ID)

	s.Require().NoError(err)
	s.Empty(players)
}

func (s *testSuite) TestARoundNeverOpenedIsNone() {
	_, err := s.query.Players(s.T().Context(), "season-1/finale")

	s.Require().ErrorIs(err, round_players_query.ErrNoRound)
}

package my_season_query_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_my_season_handler/my_season_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/postgres_contribution_store"
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

func seasonZero() calendar.Calendar {
	return calendar.New(calendar.Config{List: []calendar.Entry{
		{Number: 0, EndsAt: seasonZeroEnds, Finale: 2 * time.Hour},
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

func (s *testSuite) take(n int, country standings.Country, tiles int) {
	for range tiles {
		s.Require().NoError(s.store.RecordTake(s.T().Context(), 0, standings.Take{Account: account(n), Country: country}))
	}
}

func (s *testSuite) player(n int, country standings.Country, tiles int) {
	s.take(n, country, tiles)
	s.authors.named[account(n)] = &playerv1.Author{AccountId: account(n).String(), Name: fmt.Sprintf("player_%d", n)}
}

func (s *testSuite) guest(n int, country standings.Country, tiles int) {
	s.take(n, country, tiles)
	s.authors.named[account(n)] = &playerv1.Author{AccountId: account(n).String(), Name: fmt.Sprintf("guest_%06x", n), Guest: true}
}

func (s *testSuite) query() *my_season_query.PostgresQuery {
	return my_season_query.NewPostgresQuery(s.db, s.authors, seasonZero(), s.clock)
}

func (s *testSuite) mySeason(n int) *seasonsv1.GetMySeasonResponse {
	res, err := s.query().MySeason(s.T().Context(), account(n))
	s.Require().NoError(err)
	return res
}

func (s *testSuite) TestTheRanksCountThePlayersAboveOnTheWholeMapAndInTheMainFlag() {
	s.player(1, "fr", 5)
	s.player(2, "fr", 7)
	s.player(3, "de", 9)
	s.guest(4, "fr", 8)
	s.player(5, "fr", 5)
	s.player(6, "de", 3)
	s.take(7, "fr", 9)

	s.True(proto.Equal(&seasonsv1.GetMySeasonResponse{CountryId: "fr", Tiles: 5, GlobalRank: 3, CountryRank: 2}, s.mySeason(1)),
		s.mySeason(1))
	s.Equal(uint32(3), s.mySeason(5).GetGlobalRank(), "a tie shares the rank")
}

func (s *testSuite) TestTheBestPlayerIsFirstEverywhere() {
	s.player(1, "fr", 5)
	s.player(2, "de", 3)

	s.True(proto.Equal(&seasonsv1.GetMySeasonResponse{CountryId: "fr", Tiles: 5, GlobalRank: 1, CountryRank: 1}, s.mySeason(1)))
	s.Equal(uint32(1), s.mySeason(2).GetCountryRank())
}

func (s *testSuite) TestTheRanksCountPastAPage() {
	for n := range 520 {
		s.player(1000+n, "de", 2)
	}
	s.player(1, "fr", 1)

	mine := s.mySeason(1)

	s.Equal(uint32(521), mine.GetGlobalRank())
	s.Equal(uint32(1), mine.GetCountryRank())
}

func (s *testSuite) TestAGuestReadsItsTilesAndNoRank() {
	s.guest(1, "fr", 4)
	s.take(1, "de", 1)
	s.player(2, "fr", 9)

	s.True(proto.Equal(&seasonsv1.GetMySeasonResponse{CountryId: "fr", Tiles: 4}, s.mySeason(1)), s.mySeason(1))
}

func (s *testSuite) TestAnAccountThePlayerModuleCannotNameHasNoRank() {
	s.take(1, "fr", 4)

	s.True(proto.Equal(&seasonsv1.GetMySeasonResponse{CountryId: "fr", Tiles: 4}, s.mySeason(1)))
}

func (s *testSuite) TestAnAccountThatTookNothingReadsNothingAndAsksNobody() {
	s.player(2, "fr", 4)

	s.True(proto.Equal(&seasonsv1.GetMySeasonResponse{}, s.mySeason(1)))
	s.Zero(s.authors.asked)
}

func (s *testSuite) TestNoSeasonIsAnEmptyAnswer() {
	s.player(1, "fr", 4)
	s.clock.Advance(48 * time.Hour)

	s.True(proto.Equal(&seasonsv1.GetMySeasonResponse{}, s.mySeason(1)))
}

func (s *testSuite) TestAFailureToNameIsAnError() {
	s.player(1, "fr", 4)
	refused := errors.New("the player module is down")
	s.authors.err = refused

	_, err := s.query().MySeason(s.T().Context(), account(1))

	s.Require().ErrorIs(err, refused)
}

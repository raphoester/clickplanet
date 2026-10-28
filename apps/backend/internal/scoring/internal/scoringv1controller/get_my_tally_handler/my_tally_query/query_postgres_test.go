package my_tally_query_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_my_tally_handler/my_tally_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings/postgres_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type fakeAuthors struct {
	mu    sync.Mutex
	named map[standings.AccountID]*playerv1.Author
	asked int
	err   error
}

func (f *fakeAuthors) Authors(
	_ context.Context,
	accounts []standings.AccountID,
) (map[standings.AccountID]*playerv1.Author, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
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

type testSuite struct {
	suite.Suite

	db      *cppg.Postgres
	store   *postgres_contribution_store.Store
	authors *fakeAuthors
	taken   uint64
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "scoring", migrations.FS)
	s.store = postgres_contribution_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.authors = &fakeAuthors{named: map[standings.AccountID]*playerv1.Author{}}
}

func account(n int) standings.AccountID {
	return standings.AccountID{0: 0x01, 14: byte(n >> 8), 15: byte(n)}
}

func (s *testSuite) take(n int, country standings.Country, tiles int) {
	for range tiles {
		s.taken++
		s.Require().NoError(s.store.RecordTake(s.T().Context(), "season-0", standings.Take{Account: account(n), Country: country, Position: s.taken}))
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

func (s *testSuite) query() *my_tally_query.PostgresQuery {
	return my_tally_query.NewPostgresQuery(s.db, s.authors, cpcountries.New())
}

func (s *testSuite) mySeason(n int, country string) *scoringv1.GetMyTallyResponse {
	res, err := s.query().MyTally(s.T().Context(), "season-0", account(n), country)
	s.Require().NoError(err)
	return res
}

func (s *testSuite) TestTheGlobalRankCountsThePlayersAboveByTheTilesOfTheirMainFlag() {
	s.player(1, "fr", 5)
	s.player(2, "fr", 7)
	s.player(3, "de", 9)
	s.guest(4, "fr", 8)
	s.player(5, "fr", 5)
	s.player(6, "de", 3)
	s.take(7, "fr", 9)
	s.take(6, "it", 4)

	s.True(proto.Equal(&scoringv1.GetMyTallyResponse{CountryId: "fr", Tiles: 5, GlobalRank: 3}, s.mySeason(1, "")),
		s.mySeason(1, ""))
	s.Equal(uint32(3), s.mySeason(5, "").GetGlobalRank(), "a tie shares the rank")
}

func (s *testSuite) TestTheCountryRankCountsEveryPlayerAboveByTheTilesTakenForThatCountry() {
	s.player(1, "bg", 74)
	s.take(1, "fr", 3)
	s.player(2, "fr", 2)
	s.player(3, "de", 9)
	s.take(3, "fr", 5)
	s.guest(4, "fr", 8)
	s.take(5, "fr", 9)
	s.player(6, "fr", 3)

	s.True(proto.Equal(&scoringv1.GetMyTallyResponse{
		CountryId: "bg", Tiles: 74, GlobalRank: 1, CountryTiles: 3, CountryRank: 2,
	}, s.mySeason(1, "fr")), s.mySeason(1, "fr"))
	s.Equal(uint32(2), s.mySeason(6, "fr").GetCountryRank(), "a tie shares the rank")
	s.Equal(uint32(4), s.mySeason(2, "fr").GetCountryRank())
}

func (s *testSuite) TestTheMainFlagIsACountryLikeAnyOther() {
	s.player(1, "fr", 5)
	s.player(2, "de", 9)
	s.take(2, "fr", 6)

	s.True(proto.Equal(&scoringv1.GetMyTallyResponse{
		CountryId: "fr", Tiles: 5, GlobalRank: 2, CountryTiles: 5, CountryRank: 2,
	}, s.mySeason(1, "fr")), s.mySeason(1, "fr"))
}

func (s *testSuite) TestACountryTheCallerTookNothingForHasNoTilesAndNoRank() {
	s.player(1, "fr", 5)
	s.player(2, "de", 9)

	s.True(proto.Equal(&scoringv1.GetMyTallyResponse{CountryId: "fr", Tiles: 5, GlobalRank: 2}, s.mySeason(1, "de")),
		s.mySeason(1, "de"))
}

func (s *testSuite) TestTheBestPlayerIsFirstEverywhere() {
	s.player(1, "fr", 5)
	s.player(2, "de", 3)

	s.True(proto.Equal(&scoringv1.GetMyTallyResponse{
		CountryId: "fr", Tiles: 5, GlobalRank: 1, CountryTiles: 5, CountryRank: 1,
	}, s.mySeason(1, "fr")))
	s.Equal(uint32(1), s.mySeason(2, "de").GetCountryRank())
}

func (s *testSuite) TestTheCallerWearsItsTitle() {
	og := &playerv1.Title{Id: "og", Name: "OG"}
	s.player(1, "fr", 5)
	s.authors.named[account(1)].WornTitle = og

	s.True(proto.Equal(
		&scoringv1.GetMyTallyResponse{CountryId: "fr", Tiles: 5, GlobalRank: 1, CountryTiles: 5, CountryRank: 1, WornTitle: og},
		s.mySeason(1, "fr"),
	), s.mySeason(1, "fr"))
}

func (s *testSuite) TestTheRanksCountPastAPage() {
	for n := range 520 {
		s.player(1000+n, "de", 2)
		s.take(1000+n, "fr", 2)
	}
	s.player(1, "fr", 1)

	mine := s.mySeason(1, "fr")

	s.Equal(uint32(521), mine.GetGlobalRank())
	s.Equal(uint32(521), mine.GetCountryRank())
}

func (s *testSuite) TestAGuestReadsItsTilesAndNoRank() {
	s.guest(1, "fr", 4)
	s.authors.named[account(1)].WornTitle = &playerv1.Title{Id: "og", Name: "OG"}
	s.take(1, "de", 1)
	s.player(2, "fr", 9)

	s.True(proto.Equal(&scoringv1.GetMyTallyResponse{CountryId: "fr", Tiles: 4, CountryTiles: 1}, s.mySeason(1, "de")),
		s.mySeason(1, "de"))
}

func (s *testSuite) TestAnAccountThePlayerModuleCannotNameHasNoRank() {
	s.take(1, "fr", 4)

	s.True(proto.Equal(&scoringv1.GetMyTallyResponse{CountryId: "fr", Tiles: 4, CountryTiles: 4}, s.mySeason(1, "fr")))
}

func (s *testSuite) TestAnAccountThatTookNothingReadsNothingAndAsksNobody() {
	s.player(2, "fr", 4)

	s.True(proto.Equal(&scoringv1.GetMyTallyResponse{}, s.mySeason(1, "fr")))
	s.Zero(s.authors.asked)
}

func (s *testSuite) TestAnotherBookIsAnEmptyAnswer() {
	s.player(1, "fr", 4)

	res, err := s.query().MyTally(s.T().Context(), "season-1", account(1), "fr")
	s.Require().NoError(err)
	s.True(proto.Equal(&scoringv1.GetMyTallyResponse{}, res))
}

func (s *testSuite) TestACountryThatIsNotOneIsRefusedAndReadsNothing() {
	s.player(1, "fr", 4)

	_, err := s.query().MyTally(s.T().Context(), "season-0", account(1), "zz")

	s.Require().ErrorIs(err, my_tally_query.ErrUnknownCountry)
	s.Zero(s.authors.asked)
}

func (s *testSuite) TestAFailureToNameIsAnError() {
	s.player(1, "fr", 4)
	refused := errors.New("the player module is down")
	s.authors.err = refused

	_, err := s.query().MyTally(s.T().Context(), "season-0", account(1), "fr")

	s.Require().ErrorIs(err, refused)
}

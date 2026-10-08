package fronts_query_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/fronts"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/fronts/postgres_front_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_fronts_handler/fronts_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db     *cppg.Postgres
	fronts *postgres_front_store.Store
}

var ada = players.AccountID{15: 1}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "player", migrations.FS)
	s.fronts = postgres_front_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) took(account players.AccountID, country, previous fronts.Country, tiles int) {
	take, err := fronts.NewTake(account, country, previous)
	s.Require().NoError(err)
	for range tiles {
		s.Require().NoError(s.fronts.RecordTake(s.T().Context(), take))
	}
}

func (s *testSuite) TestThePlayerReadsEveryCountryItPlaysForAndAgainstMostTilesFirst() {
	s.took(ada, "fr", "de", 5)
	s.took(ada, "be", "es", 1)
	s.took(ada, "it", "pt", 1)
	s.took(ada, "nl", "ch", 1)
	s.took(ada, "fr", "it", 2)
	s.took(players.AccountID{15: 2}, "de", "fr", 9)

	answer, err := fronts_query.NewPostgresQuery(s.db).Fronts(s.T().Context(), ada)

	s.Require().NoError(err)
	s.True(proto.Equal(&playerv1.GetFrontsResponse{
		PlaysFor: []*playerv1.CountryTiles{
			{CountryId: "fr", Tiles: 7}, {CountryId: "be", Tiles: 1}, {CountryId: "it", Tiles: 1}, {CountryId: "nl", Tiles: 1},
		},
		PlaysAgainst: []*playerv1.CountryTiles{
			{CountryId: "de", Tiles: 5},
			{CountryId: "it", Tiles: 2},
			{CountryId: "ch", Tiles: 1},
			{CountryId: "es", Tiles: 1},
			{CountryId: "pt", Tiles: 1},
		},
	}, answer), "%v", answer)
}

func (s *testSuite) TestAPlayerThatTookNothingReadsNothing() {
	answer, err := fronts_query.NewPostgresQuery(s.db).Fronts(s.T().Context(), ada)

	s.Require().NoError(err)
	s.Empty(answer.GetPlaysFor())
	s.Empty(answer.GetPlaysAgainst())
}

func (s *testSuite) TestAStoreFailureIsAnError() {
	ctx, cancel := context.WithCancel(s.T().Context())
	cancel()

	_, err := fronts_query.NewPostgresQuery(s.db).Fronts(ctx, ada)

	s.Error(err)
}

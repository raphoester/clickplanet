package stats_query_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/postgres_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_stats_handler/stats_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db      *cppg.Postgres
	players *postgres_player_store.Store
}

var (
	ada    = players.AccountID{15: 1}
	monday = time.Date(2026, 9, 14, 20, 0, 0, 0, time.UTC)
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "player", migrations.FS)
	s.players = postgres_player_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) stats(clock cptime.Clock) *playerv1.Stats {
	answer, err := stats_query.NewPostgresQuery(s.db, clock).Stats(s.T().Context(), ada)
	s.Require().NoError(err)
	return answer.GetStats()
}

func (s *testSuite) TestAnAccountThatNeverTookATileHasEmptyStats() {
	s.True(proto.Equal(&playerv1.Stats{}, s.stats(cptime.NewFixedClock(monday))))
}

func (s *testSuite) TestTheStreakIsReadAsOfTodayInUTC() {
	s.Require().NoError(s.players.RecordTake(s.T().Context(), ada, monday))
	s.Require().NoError(s.players.RecordTake(s.T().Context(), ada, monday.Add(24*time.Hour)))
	clock := cptime.NewFixedClock(monday.Add(48 * time.Hour))

	s.True(proto.Equal(&playerv1.Stats{TilesTaken: 2, StreakCurrent: 2, StreakBest: 2, StreakLastDay: "2026-09-15"}, s.stats(clock)),
		"Wednesday can still extend it")

	clock.Advance(24 * time.Hour)
	s.True(proto.Equal(&playerv1.Stats{TilesTaken: 2, StreakCurrent: 0, StreakBest: 2, StreakLastDay: "2026-09-15"}, s.stats(clock)),
		"Thursday: Wednesday went by with no take")
}

func (s *testSuite) TestAnAccountThatOnlyPostedHasNoStreakDay() {
	s.Require().NoError(s.players.RecordMessage(s.T().Context(), ada))

	s.True(proto.Equal(&playerv1.Stats{}, s.stats(cptime.NewFixedClock(monday))))
}

func (s *testSuite) TestAFailedReadIsAnError() {
	ctx, cancel := context.WithCancel(s.T().Context())
	cancel()

	_, err := stats_query.NewPostgresQuery(s.db, cptime.NewFixedClock(monday)).Stats(ctx, ada)

	s.Error(err, "not empty stats")
}

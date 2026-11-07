package raid_mode_query_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_raid_mode_handler/raid_mode_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids/postgres_raid_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

var now = time.Date(2026, 11, 7, 20, 0, 0, 0, time.UTC)

type testSuite struct {
	suite.Suite

	db    *cppg.Postgres
	clock *cptime.FixedClock
	query *raid_mode_query.PostgresQuery
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "chat", migrations.FS)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.clock = cptime.NewFixedClock(now)
	s.query = raid_mode_query.NewPostgresQuery(s.db, s.clock)
}

func (s *testSuite) started(duration time.Duration, reason moderation.Reason) raids.RaidMode {
	mode, err := raids.Start(now, duration, reason)
	s.Require().NoError(err)
	s.Require().NoError(postgres_raid_store.New(s.db).Save(s.T().Context(), mode))
	return mode
}

func (s *testSuite) TestNoRaidModeIsAnsweredWithNone() {
	res, err := s.query.RaidMode(s.T().Context())

	s.Require().NoError(err)
	s.Nil(res.GetRaidMode())
}

func (s *testSuite) TestTheRaidModeThatRunsIsAnsweredAsTheBookReadsIt() {
	mode := s.started(time.Hour, "a raid of guests")

	res, err := s.query.RaidMode(s.T().Context())

	s.Require().NoError(err)
	s.Equal(timestamppb.New(now), res.GetRaidMode().GetStartedAt())
	s.Equal(timestamppb.New(now.Add(time.Hour)), res.GetRaidMode().GetEndsAt())
	s.Equal("a raid of guests", res.GetRaidMode().GetReason())
	s.True(mode.RunsAt(s.clock.Now()), "the query and the rule agree")
}

func (s *testSuite) TestARaidModeThatEndedIsAnsweredWithNone() {
	mode := s.started(time.Hour, "")
	s.clock.Advance(time.Hour)

	res, err := s.query.RaidMode(s.T().Context())

	s.Require().NoError(err)
	s.Nil(res.GetRaidMode())
	s.False(mode.RunsAt(s.clock.Now()), "the query and the rule agree")
}

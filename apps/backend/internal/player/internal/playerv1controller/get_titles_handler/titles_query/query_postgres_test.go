package titles_query_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/postgres_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_titles_handler/titles_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/inprocess_title_catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/postgres_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing/postgres_worn_title_store"
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
	titles  *postgres_title_store.Store
	worn    *postgres_worn_title_store.Store
}

var (
	ada    = players.AccountID{15: 1}
	monday = time.Date(2026, 9, 14, 20, 0, 0, 0, time.UTC)
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "player", migrations.FS)
	s.players = postgres_player_store.New(s.db)
	s.titles = postgres_title_store.New(s.db)
	s.worn = postgres_worn_title_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) dashboard(clock cptime.Clock) *playerv1.GetTitlesResponse {
	answer, err := titles_query.NewPostgresQuery(s.db, inprocess_title_catalog.New(titles.NewCatalog()), clock).Titles(s.T().Context(), ada)
	s.Require().NoError(err)
	return answer
}

func progress(dashboard *playerv1.GetTitlesResponse) map[string]uint64 {
	by := make(map[string]uint64, len(dashboard.GetTracks()))
	for _, track := range dashboard.GetTracks() {
		by[track.GetId()] = track.GetProgress()
	}
	return by
}

func (s *testSuite) TestTheDashboardMeasuresTheStreakAsOfToday() {
	s.Require().NoError(s.players.RecordTake(s.T().Context(), ada, monday))
	s.Require().NoError(s.players.RecordTake(s.T().Context(), ada, monday.Add(24*time.Hour)))
	s.Require().NoError(s.titles.Grant(s.T().Context(), titles.Holdings{ada: {"settler"}}, monday))
	clock := cptime.NewFixedClock(monday.Add(24 * time.Hour))

	dashboard := s.dashboard(clock)
	s.Equal("settler", dashboard.GetWorn().GetId())
	s.Equal(map[string]uint64{"conquest": 2, "devotion": 2, "chatter": 0}, progress(dashboard), "the streak on its second day")

	clock.Advance(72 * time.Hour)
	s.Equal(uint64(0), progress(s.dashboard(clock))["devotion"], "a streak broken since reads zero")
}

func (s *testSuite) TestTheChatterTrackCountsTheMessagesSent() {
	for range 3 {
		s.Require().NoError(s.players.RecordMessage(s.T().Context(), ada))
	}

	s.Equal(uint64(3), progress(s.dashboard(cptime.NewFixedClock(monday)))["chatter"])
}

func (s *testSuite) TestAnAccountWithNoStatsStartsEveryTrackAtZero() {
	dashboard := s.dashboard(cptime.NewFixedClock(monday))

	s.Nil(dashboard.GetWorn())
	s.Empty(dashboard.GetWearable())
	s.Equal(map[string]uint64{"conquest": 0, "devotion": 0, "chatter": 0}, progress(dashboard))
}

func (s *testSuite) TestTheDashboardWearsTheChoiceAndOffersWhatIsShown() {
	s.Require().NoError(s.titles.Grant(s.T().Context(), titles.Holdings{ada: {"og", "settler", "raider", "loyal"}}, monday))
	s.Require().NoError(s.worn.Wear(s.T().Context(), ada, "loyal", monday))

	dashboard := s.dashboard(cptime.NewFixedClock(monday))

	s.Equal("loyal", dashboard.GetWorn().GetId())
	wearable := make([]string, 0, len(dashboard.GetWearable()))
	for _, title := range dashboard.GetWearable() {
		wearable = append(wearable, title.GetId())
	}
	s.Equal([]string{"og", "raider", "loyal"}, wearable, "the highest rank of each track, not the ones below it")
}

func (s *testSuite) TestEachStepSaysItsThresholdAndWhetherItIsEarned() {
	s.Require().NoError(s.titles.Grant(s.T().Context(), titles.Holdings{ada: {"settler"}}, monday))

	conquest := s.dashboard(cptime.NewFixedClock(monday)).GetTracks()[0]

	s.Equal("conquest", conquest.GetId())
	s.Equal("Conquest", conquest.GetName())
	s.Require().Len(conquest.GetSteps(), 5)
	s.Equal("settler", conquest.GetSteps()[0].GetTitle().GetId())
	s.Equal(uint64(100), conquest.GetSteps()[0].GetThreshold())
	s.True(conquest.GetSteps()[0].GetEarned())
	s.False(conquest.GetSteps()[1].GetEarned())
}

func (s *testSuite) TestAFailedReadIsAnError() {
	ctx, cancel := context.WithCancel(s.T().Context())
	cancel()

	_, err := titles_query.NewPostgresQuery(s.db, inprocess_title_catalog.New(titles.NewCatalog()), cptime.NewFixedClock(monday)).Titles(ctx, ada)

	s.Error(err)
}

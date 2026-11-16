package level_query_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/postgres_level_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/get_level_handler/level_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/inprocess_level_scale"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/postgres_stats_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db     *cppg.Postgres
	stats  *postgres_stats_store.Store
	levels *postgres_level_store.Store
}

var (
	ada    = accounts.AccountID{15: 1}
	monday = time.Date(2026, 11, 16, 20, 0, 0, 0, time.UTC)
)

func (s *testSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "progression", migrations.FS)
	s.stats = postgres_stats_store.New(s.db)
	s.levels = postgres_level_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) level(account accounts.AccountID) *progressionv1.GetLevelResponse {
	answer, err := level_query.NewPostgresQuery(s.db, inprocess_level_scale.New(levels.Scale{})).Level(s.T().Context(), account)
	s.Require().NoError(err)
	return answer
}

func (s *testSuite) took(account accounts.AccountID) {
	s.Require().NoError(s.stats.Record(s.T().Context(), account, stats.TakeAt(0, 0, monday)))
	s.Require().NoError(s.stats.Record(s.T().Context(), account,
		stats.MissionsDone(map[stats.MissionKind]stats.MissionTally{"take_from": {Done: 1, Goals: 81}, "catch_boxes": {Done: 1, Goals: 3}})))
}

func (s *testSuite) TestALinkedAccountReadsItsLevelKeptWithTheXPItsStatsGiveNow() {
	s.took(ada)
	s.Require().NoError(s.levels.RaiseAll(s.T().Context(), levels.ByAccount{ada: 4}, monday))

	level := s.level(ada).GetLevel()

	s.True(proto.Equal(&progressionv1.Level{Level: 4, Xp: 1 + 100 + 121 + 150, LevelXp: 378, NextLevelXp: 896}, level), "%v", level)
}

func (s *testSuite) TestALinkedAccountWithNoLevelKeptReadsNone() {
	s.took(ada)

	s.Nil(s.level(ada).GetLevel())
}

func (s *testSuite) TestAGuestReadsNoLevelWhateverItDid() {
	s.took(ada)

	answer := s.level(ada)

	s.Nil(answer.GetLevel())
	s.NotEmpty(answer.GetMissionRates(), "the rules are the game's, not the account's")
}

func (s *testSuite) TestALevelKeptWithNoStatsIsOnItsBarWithNoXP() {
	s.Require().NoError(s.levels.RaiseAll(s.T().Context(), levels.ByAccount{ada: 1}, monday))

	s.True(proto.Equal(&progressionv1.Level{Level: 1, NextLevelXp: 14}, s.level(ada).GetLevel()))
}

func (s *testSuite) TestEveryAnswerTellsWhatEachKindOfMissionPays() {
	rates := s.level(ada).GetMissionRates()

	s.Len(rates, len(missionsv1.MissionKind_name)-1)
	s.True(proto.Equal(&progressionv1.MissionRate{Kind: missionsv1.MissionKind_MISSION_KIND_TAKE_FROM, Xp: 3, Per: 2}, rates[1]))
}

func (s *testSuite) TestEveryAnswerTellsTheLevelEachKindUnlocksAt() {
	unlocks := s.level(ada).GetUnlocks()

	s.Require().Len(unlocks, 4)
	s.True(proto.Equal(&progressionv1.Unlock{Kind: planetv1.BonusKind_BONUS_KIND_ENCLOSE_CLICKS, Level: 4}, unlocks[0]))
	s.True(proto.Equal(&progressionv1.Unlock{Kind: planetv1.BonusKind_BONUS_KIND_BOMB, Level: 5}, unlocks[1]))
	s.True(proto.Equal(&progressionv1.Unlock{Kind: planetv1.BonusKind_BONUS_KIND_VIRUS, Level: 6}, unlocks[2]))
	s.True(proto.Equal(&progressionv1.Unlock{Kind: planetv1.BonusKind_BONUS_KIND_MINE, Level: 8}, unlocks[3]))
}

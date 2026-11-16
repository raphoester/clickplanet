package admin_progress_query_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/postgres_level_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/admin_get_progress_handler/admin_progress_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/inprocess_level_scale"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/inprocess_title_catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/progressionmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/postgres_stats_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles/postgres_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/wearing/postgres_worn_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type fakeFronts struct {
	answer *progressionv1.GetFrontsResponse
	err    error
}

func (f *fakeFronts) Fronts(context.Context, accounts.AccountID) (*progressionv1.GetFrontsResponse, error) {
	return f.answer, f.err
}

type testSuite struct {
	suite.Suite

	db     *cppg.Postgres
	stats  *postgres_stats_store.Store
	titles *postgres_title_store.Store
	worn   *postgres_worn_title_store.Store
	levels *postgres_level_store.Store
	fronts *fakeFronts
}

var (
	ada    = accounts.AccountID{15: 1}
	monday = time.Date(2026, 9, 14, 20, 0, 0, 0, time.UTC)
)

func (s *testSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "progression", migrations.FS)
	s.stats = postgres_stats_store.New(s.db)
	s.titles = postgres_title_store.New(s.db)
	s.worn = postgres_worn_title_store.New(s.db)
	s.levels = postgres_level_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.fronts = &fakeFronts{answer: &progressionv1.GetFrontsResponse{}}
}

func (s *testSuite) query() *admin_progress_query.PostgresQuery {
	return admin_progress_query.NewPostgresQuery(s.db, inprocess_title_catalog.New(titles.NewCatalog()),
		inprocess_level_scale.New(levels.Scale{}), s.fronts, cptime.NewFixedClock(monday))
}

func (s *testSuite) progress(account accounts.AccountID) *progressionv1.AdminGetProgressResponse {
	answer, err := s.query().Progress(s.T().Context(), account)
	s.Require().NoError(err)
	return answer
}

func (s *testSuite) TestEverythingTheGameKeepsIsRead() {
	ctx := s.T().Context()
	s.Require().NoError(s.stats.Record(ctx, ada, stats.TakeAt(0, 0, monday.Add(-24*time.Hour))))
	s.Require().NoError(s.stats.Record(ctx, ada, stats.TakeAt(1, 0, monday)))
	s.Require().NoError(s.stats.Record(ctx, ada, stats.MessageAt(monday)))
	s.Require().NoError(s.titles.Grant(ctx, titles.Holdings{ada: {"settler", "og"}}, monday))
	s.Require().NoError(s.worn.Wear(ctx, ada, "og", monday))

	og, _ := titles.NewCatalog().StandingOf("og")
	settler, _ := titles.NewCatalog().StandingOf("settler")
	s.True(proto.Equal(&progressionv1.AdminGetProgressResponse{
		Stats:        &progressionv1.Stats{TilesTaken: 2, StreakCurrent: 2, StreakBest: 2, StreakLastDay: "2026-09-14"},
		MessagesSent: 1,
		DaysPlayed:   2,
		Xp:           &progressionv1.XpBreakdown{Total: 202, TilesTaken: 2, DaysPlayed: 200},
		Titles:       progressionmessage.Titles([]titles.Standing{og, settler}),
		WornTitle:    progressionmessage.Title(og),
	}, s.progress(ada)), "%v", s.progress(ada))
}

func (s *testSuite) TestAnAccountWithNoNameplateIsReadAsKept() {
	s.Require().NoError(s.stats.Record(s.T().Context(), ada, stats.TakeAt(2, 0, monday)))

	s.Equal(uint32(1), s.progress(ada).GetStats().GetStreakCurrent(), "the game shows a guest no streak, and the operator what is kept")
}

func (s *testSuite) TestTheDaysPlayedAreTheOnesKept() {
	ctx := s.T().Context()
	s.Require().NoError(s.stats.Record(ctx, ada, stats.TakeAt(0, 0, monday.Add(-72*time.Hour))))
	s.Require().NoError(s.stats.Record(ctx, ada, stats.TakeAt(1, 0, monday)))
	s.Require().NoError(s.stats.Record(ctx, ada, stats.PlayDays([]stats.Day{
		stats.DayOf(monday.Add(-10 * 24 * time.Hour)), stats.DayOf(monday.Add(-72 * time.Hour)),
	})))

	s.Equal(uint64(3), s.progress(ada).GetDaysPlayed(), "not the streak: the days need not follow each other")
}

func (s *testSuite) TestTheLevelKeptTheXPTheStatsGiveAndTheMissionsByKindAreRead() {
	ctx := s.T().Context()
	s.Require().NoError(s.stats.Record(ctx, ada, stats.TakeAt(0, 0, monday)))
	s.Require().NoError(s.stats.Record(ctx, ada, stats.BombAt(1)))
	s.Require().NoError(s.stats.Record(ctx, ada, stats.FortifyAt(2, 50)))
	s.Require().NoError(s.stats.Record(ctx, ada, stats.BonusAt(monday)))
	s.Require().NoError(s.stats.Record(ctx, ada, stats.RightAnswerAt(monday)))
	s.Require().NoError(s.stats.Record(ctx, ada, stats.MissionsDone(map[stats.MissionKind]stats.MissionTally{
		"take_for": {Done: 2, Goals: 360}, "bomb_in": {Done: 1, Goals: 1},
	})))
	_, err := s.levels.Raise(ctx, ada, 4, monday)
	s.Require().NoError(err)

	progress := s.progress(ada)

	s.Equal(uint32(4), progress.GetLevel(), "the level kept, not the one the XP reaches")
	s.True(proto.Equal(&progressionv1.XpBreakdown{
		Total: 1 + 100 + 360 + 150 + 50 + 30 + 30 + 10, TilesTaken: 1, DaysPlayed: 100, MissionsDone: 360 + 150, BombsDropped: 50,
		BoxesCaught: 30, RightAnswers: 30, TilesFortified: 10,
	}, progress.GetXp()), "%v", progress.GetXp())
	s.True(proto.Equal(&progressionv1.AdminGetProgressResponse{MissionsByKind: []*progressionv1.MissionTally{
		{Kind: "bomb_in", Done: 1, Goals: 1}, {Kind: "take_for", Done: 2, Goals: 360},
	}}, &progressionv1.AdminGetProgressResponse{MissionsByKind: progress.GetMissionsByKind()}), "%v", progress.GetMissionsByKind())
}

func (s *testSuite) TestAGuestHasXPAndNoLevel() {
	s.Require().NoError(s.stats.Record(s.T().Context(), ada, stats.TakeAt(0, 0, monday)))

	s.Zero(s.progress(ada).GetLevel())
	s.Equal(uint64(101), s.progress(ada).GetXp().GetTotal(), "what the operator reads is the stats' XP")
}

func (s *testSuite) TestAnAccountTheGameKeepsNothingAboutIsZeros() {
	s.True(proto.Equal(&progressionv1.AdminGetProgressResponse{
		Stats: &progressionv1.Stats{}, Titles: []*progressionv1.Title{}, Xp: &progressionv1.XpBreakdown{},
	}, s.progress(accounts.AccountID{15: 9})))
}

func (s *testSuite) TestEveryCountryThePlayerPlaysForAndAgainstIsRead() {
	countries := func(codes ...string) []*progressionv1.CountryTiles {
		tiles := make([]*progressionv1.CountryTiles, 0, len(codes))
		for i, code := range codes {
			tiles = append(tiles, &progressionv1.CountryTiles{CountryId: code, Tiles: uint64(10 - i)}) //nolint:gosec // a few countries.
		}
		return tiles
	}
	s.fronts.answer = &progressionv1.GetFrontsResponse{PlaysFor: countries("fr", "be"), PlaysAgainst: countries("de", "es", "it", "pt")}

	progress := s.progress(ada)

	s.True(proto.Equal(&progressionv1.AdminGetProgressResponse{PlaysFor: countries("fr", "be"), PlaysAgainst: countries("de", "es", "it", "pt")},
		&progressionv1.AdminGetProgressResponse{PlaysFor: progress.GetPlaysFor(), PlaysAgainst: progress.GetPlaysAgainst()}),
		"not the public card's top three: %v", progress)
}

func (s *testSuite) TestAFailureToAskPlanetIsAnError() {
	s.fronts.err = errors.New("down")

	_, err := s.query().Progress(s.T().Context(), ada)

	s.Require().ErrorIs(err, s.fronts.err)
}

package progress_query_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/departures"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/departures/postgres_departure_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/postgres_level_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/nameplates/postgres_nameplate_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/get_progress_handler/progress_query"
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

	db         *cppg.Postgres
	stats      *postgres_stats_store.Store
	nameplates *postgres_nameplate_store.Store
	titles     *postgres_title_store.Store
	worn       *postgres_worn_title_store.Store
	levels     *postgres_level_store.Store
	fronts     *fakeFronts
	clock      *cptime.FixedClock
}

var (
	ada    = accounts.AccountID{15: 1}
	monday = time.Date(2026, 9, 14, 20, 0, 0, 0, time.UTC)
)

func (s *testSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "progression", migrations.FS)
	s.stats = postgres_stats_store.New(s.db)
	s.nameplates = postgres_nameplate_store.New(s.db)
	s.titles = postgres_title_store.New(s.db)
	s.worn = postgres_worn_title_store.New(s.db)
	s.levels = postgres_level_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.fronts = &fakeFronts{answer: &progressionv1.GetFrontsResponse{}}
	s.clock = cptime.NewFixedClock(monday)
	s.Require().NoError(s.nameplates.Keep(s.T().Context(), ada))
}

func (s *testSuite) query() *progress_query.PostgresQuery {
	return progress_query.NewPostgresQuery(
		s.db, inprocess_title_catalog.New(titles.NewCatalog()), inprocess_level_scale.New(levels.Scale{}), s.fronts, s.clock,
	)
}

func (s *testSuite) progress(account accounts.AccountID) *progressionv1.Progress {
	answer, err := s.query().Progress(s.T().Context(), account)
	s.Require().NoError(err)
	return answer.GetProgress()
}

func (s *testSuite) TestAPlayersProgressIsItsStatsAsOfToday() {
	s.Require().NoError(s.stats.Record(s.T().Context(), ada, stats.TakeAt(0, 0, monday.Add(-24*time.Hour))))
	s.Require().NoError(s.stats.Record(s.T().Context(), ada, stats.TakeAt(1, 0, monday)))

	s.True(proto.Equal(&progressionv1.Progress{
		Stats:  &progressionv1.Stats{TilesTaken: 2, StreakCurrent: 2, StreakBest: 2, StreakLastDay: "2026-09-14"},
		Titles: []*progressionv1.Title{},
	}, s.progress(ada)), "%v", s.progress(ada))
}

func (s *testSuite) TestAPlayerThatTookNothingHasEmptyProgress() {
	s.True(proto.Equal(&progressionv1.Progress{Stats: &progressionv1.Stats{}, Titles: []*progressionv1.Title{}}, s.progress(ada)))
}

func (s *testSuite) TestALevelKeptIsShownOnItsBarWithTheXPTheStatsGiveNow() {
	s.Require().NoError(s.stats.Record(s.T().Context(), ada, stats.TakeAt(0, 0, monday)))
	s.Require().NoError(s.stats.Record(s.T().Context(), ada,
		stats.MissionsDone(map[stats.MissionKind]stats.MissionTally{"take_from": {Done: 1, Goals: 81}})))
	s.Require().NoError(s.levels.RaiseAll(s.T().Context(), levels.ByAccount{ada: 3}, monday))

	level := s.progress(ada).GetLevel()

	s.True(proto.Equal(&progressionv1.Level{Level: 3, Xp: 1 + 100 + 121, LevelXp: 112, NextLevelXp: 378}, level), "%v", level)
}

func (s *testSuite) TestAPlayerWithNoLevelKeptShowsNoneWhateverItsStats() {
	s.Require().NoError(s.stats.Record(s.T().Context(), ada, stats.TakeAt(0, 0, monday)))

	s.Nil(s.progress(ada).GetLevel())
}

func (s *testSuite) TestAGuestsLevelIsNeverShown() {
	guest := accounts.AccountID{15: 3}
	s.Require().NoError(s.stats.Record(s.T().Context(), guest, stats.TakeAt(3, 0, monday)))
	s.Require().NoError(s.levels.RaiseAll(s.T().Context(), levels.ByAccount{guest: 2}, monday))

	_, err := s.query().Progress(s.T().Context(), guest)

	s.ErrorIs(err, progress_query.ErrNotShown)
}

func (s *testSuite) TestTheStreakIsReadAsOfToday() {
	s.Require().NoError(s.stats.Record(s.T().Context(), ada, stats.TakeAt(5, 0, monday)))
	s.clock.Advance(48 * time.Hour)

	counted := s.progress(ada).GetStats()

	s.Zero(counted.GetStreakCurrent())
	s.Equal(uint32(1), counted.GetStreakBest())
}

func (s *testSuite) TestAGuestOrAnAccountNeverNamedIsNotShown() {
	guest := accounts.AccountID{15: 3}
	s.Require().NoError(s.stats.Record(s.T().Context(), guest, stats.TakeAt(3, 0, monday)))

	for _, account := range []accounts.AccountID{guest, {15: 5}} {
		_, err := s.query().Progress(s.T().Context(), account)

		s.ErrorIs(err, progress_query.ErrNotShown, "%s", account)
	}
}

func (s *testSuite) TestAnAccountAskedToBeDeletedIsNotShownUntilItComesBack() {
	store := postgres_departure_store.New(s.db)
	decide := func(decision departures.Decision) {
		s.Require().NoError(store.Change(s.T().Context(), ada,
			func(d departures.Departure) departures.Departure { return d.Decided(decision) }))
	}
	decide(departures.Leaves(monday))

	_, err := s.query().Progress(s.T().Context(), ada)
	s.Require().ErrorIs(err, progress_query.ErrNotShown)

	decide(departures.Returns(monday.Add(time.Hour)))

	s.NotNil(s.progress(ada))
}

func (s *testSuite) TestThePlayerShowsItsBestOfEachTrackAndWearsTheFirst() {
	s.Require().NoError(s.titles.Grant(s.T().Context(), titles.Holdings{ada: {"loyal", "raider", "settler", "og"}}, monday))

	progress := s.progress(ada)

	raider, _ := titles.NewCatalog().StandingOf("raider")
	loyal, _ := titles.NewCatalog().StandingOf("loyal")
	og, _ := titles.NewCatalog().StandingOf("og")
	s.True(proto.Equal(progressionmessage.Title(raider), progress.GetWornTitle()))
	want := progressionmessage.Titles([]titles.Standing{og, raider, loyal})
	s.Require().Len(progress.GetTitles(), len(want))
	for i := range want {
		s.True(proto.Equal(want[i], progress.GetTitles()[i]), want[i].GetId())
	}
}

func (s *testSuite) TestThePlayerWearsTheTitleItChose() {
	s.Require().NoError(s.titles.Grant(s.T().Context(), titles.Holdings{ada: {"loyal", "settler", "og"}}, monday))
	s.Require().NoError(s.worn.Wear(s.T().Context(), ada, "loyal", monday))

	s.Equal("loyal", s.progress(ada).GetWornTitle().GetId())
}

func (s *testSuite) TestATitleTheCatalogNoLongerHasIsNotShown() {
	s.Require().NoError(s.titles.Grant(s.T().Context(), titles.Holdings{ada: {"retired", "og"}}, monday))

	shown := s.progress(ada).GetTitles()

	s.Require().Len(shown, 1, "the next reconciliation revokes it")
	s.Equal("og", shown[0].GetId())
}

func (s *testSuite) TestAnybodyReadsTheTopThreeCountriesThePlayerPlaysForAndAgainst() {
	countries := func(codes ...string) []*progressionv1.CountryTiles {
		tiles := make([]*progressionv1.CountryTiles, 0, len(codes))
		for i, code := range codes {
			tiles = append(tiles, &progressionv1.CountryTiles{CountryId: code, Tiles: uint64(10 - i)}) //nolint:gosec // a few countries.
		}
		return tiles
	}
	s.fronts.answer = &progressionv1.GetFrontsResponse{PlaysFor: countries("fr", "be"), PlaysAgainst: countries("de", "es", "it", "pt")}

	progress := s.progress(ada)

	s.True(proto.Equal(&progressionv1.Progress{PlaysFor: countries("fr", "be"), PlaysAgainst: countries("de", "es", "it")},
		&progressionv1.Progress{PlaysFor: progress.GetPlaysFor(), PlaysAgainst: progress.GetPlaysAgainst()}), "%v", progress)
}

func (s *testSuite) TestAFailureToAskForTheCountriesIsAnError() {
	s.fronts.err = errors.New("planet is down")

	_, err := s.query().Progress(s.T().Context(), ada)

	s.Require().ErrorIs(err, s.fronts.err)
	s.NotErrorIs(err, progress_query.ErrNotShown)
}

func (s *testSuite) TestAStoreFailureIsNotNotShown() {
	ctx, cancel := context.WithCancel(s.T().Context())
	cancel()

	_, err := s.query().Progress(ctx, ada)

	s.Require().Error(err)
	s.NotErrorIs(err, progress_query.ErrNotShown)
}

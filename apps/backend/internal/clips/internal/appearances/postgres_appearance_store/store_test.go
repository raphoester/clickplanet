package postgres_appearance_store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/appearances"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/appearances/postgres_appearance_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/get_appearances_handler/appearances_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// The store keeps, and the query reads what it kept: one suite, one container.
func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db    *cppg.Postgres
	store *postgres_appearance_store.Store
	query *appearances_query.PostgresQuery
}

var now = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "clips", migrations.FS)
	s.store = postgres_appearance_store.New(s.db)
	s.query = appearances_query.NewPostgresQuery(s.db, cptime.NewFixedClock(now))
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) account(last string) casts.AccountID {
	id, err := casts.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a" + last)
	s.Require().NoError(err)
	return id
}

func (s *testSuite) keep(post string, at time.Time, platforms []appearances.Platform, accounts ...casts.AccountID) {
	kept, err := appearances.PostOf(appearances.PostID(post), "FRANCE VS TURKEY", platforms, at)
	s.Require().NoError(err)
	s.Require().NoError(s.store.Keep(s.T().Context(), kept.AppearancesOf(accounts)))
}

func (s *testSuite) snapshot(post, platform, url string, takenAt time.Time) {
	_, err := s.db.On(s.T().Context()).ExecContext(s.T().Context(),
		`INSERT INTO snapshots (taken_at, summary) VALUES ($1, '{}') ON CONFLICT DO NOTHING`, takenAt)
	s.Require().NoError(err)
	_, err = s.db.On(s.T().Context()).ExecContext(s.T().Context(),
		`INSERT INTO snapshot_rows (taken_at, post_id, platform, document) VALUES ($1, $2, $3, jsonb_build_object('url', $4::text))`,
		takenAt, post, platform, url)
	s.Require().NoError(err)
}

func (s *testSuite) appearances(account casts.AccountID) *clipsv1.GetAppearancesResponse {
	answer, err := s.query.Appearances(s.T().Context(), account)
	s.Require().NoError(err)
	return answer
}

func (s *testSuite) TestThePostsOutThisWeekNameTheCallerNewestFirst() {
	ada, bob := s.account("01"), s.account("02")
	both := []appearances.Platform{appearances.TikTok, appearances.YouTube}
	s.keep("old", now.Add(-8*24*time.Hour), both, ada)
	s.keep("monday", now.Add(-2*24*time.Hour), both, ada, bob)
	s.keep("today", now.Add(-time.Hour), []appearances.Platform{appearances.YouTube}, ada)
	s.keep("later", now.Add(time.Hour), both, ada)

	want := &clipsv1.GetAppearancesResponse{Appearances: []*clipsv1.Appearance{
		{
			PostId: "today", Title: "FRANCE VS TURKEY", ScheduledAt: timestamppb.New(now.Add(-time.Hour)),
			Links: []*clipsv1.ClipLink{{Platform: "youtube"}},
		},
		{
			PostId: "monday", Title: "FRANCE VS TURKEY", ScheduledAt: timestamppb.New(now.Add(-2 * 24 * time.Hour)),
			Links: []*clipsv1.ClipLink{{Platform: "tiktok"}, {Platform: "youtube"}},
		},
	}}
	got := s.appearances(ada)
	s.True(proto.Equal(want, got), "one not out yet, one of last week: %v", got)
	s.Len(s.appearances(bob).GetAppearances(), 1)
}

func (s *testSuite) TestALinkIsTheNewestSnapshotsAddress() {
	ada := s.account("01")
	s.keep("monday", now.Add(-2*24*time.Hour), []appearances.Platform{appearances.TikTok, appearances.YouTube}, ada)
	s.snapshot("monday", "tiktok", "https://tiktok.example/old", now.Add(-30*time.Hour))
	s.snapshot("monday", "tiktok", "https://tiktok.example/7", now.Add(-6*time.Hour))
	s.snapshot("monday", "youtube", "", now.Add(-6*time.Hour))

	got := s.appearances(ada)
	s.True(proto.Equal(&clipsv1.Appearance{
		PostId: "monday", Title: "FRANCE VS TURKEY", ScheduledAt: timestamppb.New(now.Add(-2 * 24 * time.Hour)),
		Links: []*clipsv1.ClipLink{{Platform: "tiktok", Url: "https://tiktok.example/7"}, {Platform: "youtube"}},
	}, got.GetAppearances()[0]), "%v", got)
}

func (s *testSuite) TestAPostKeptAgainIsKeptOnce() {
	ada := s.account("01")
	s.keep("monday", now.Add(-time.Hour), []appearances.Platform{appearances.TikTok}, ada)
	s.keep("monday", now.Add(-time.Hour), []appearances.Platform{appearances.TikTok, appearances.YouTube}, ada)

	s.Len(s.appearances(ada).GetAppearances(), 1)
	s.Len(s.appearances(ada).GetAppearances()[0].GetLinks(), 2)
}

func (s *testSuite) TestTheAppearancesBeforeATimeAreFoundAndForgotten() {
	ada, bob := s.account("01"), s.account("02")
	s.keep("old", now.Add(-8*24*time.Hour), []appearances.Platform{appearances.TikTok}, ada, bob)
	s.keep("today", now.Add(-time.Hour), []appearances.Platform{appearances.TikTok}, ada)

	old, err := s.store.ScheduledBefore(s.T().Context(), appearances.ShownSince(now))
	s.Require().NoError(err)
	s.ElementsMatch([]appearances.Key{{Post: "old", Account: ada}, {Post: "old", Account: bob}}, old)

	for _, key := range old {
		s.Require().NoError(s.store.Forget(s.T().Context(), key))
	}
	left, err := s.store.ScheduledBefore(s.T().Context(), now)
	s.Require().NoError(err)
	s.Equal([]appearances.Key{{Post: "today", Account: ada}}, left)
}

func (s *testSuite) TestADeletedAccountIsInNoPost() {
	ada, bob := s.account("01"), s.account("02")
	s.keep("today", now.Add(-time.Hour), []appearances.Platform{appearances.TikTok}, ada, bob)

	s.Require().NoError(s.store.ForgetAccount(s.T().Context(), ada))

	s.Empty(s.appearances(ada).GetAppearances())
	s.Len(s.appearances(bob).GetAppearances(), 1)
}

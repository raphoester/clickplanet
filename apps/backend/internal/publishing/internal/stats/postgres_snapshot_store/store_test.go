package postgres_snapshot_store_test

import (
	"encoding/json"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/get_post_links_handler/links_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/list_rows_handler/rows_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/list_snapshots_handler/snapshots_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/stats/postgres_snapshot_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

// The store keeps, and the two queries read what it kept: one suite, one container.
func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	server    *cppg.TestServer
	db        *cppg.Postgres
	store     *postgres_snapshot_store.Store
	rows      *rows_query.PostgresQuery
	summaries *snapshots_query.PostgresQuery
}

func (s *testSuite) SetupSuite() {
	s.server = cppg.StartTestServer(s.T())
	s.db = s.server.OpenSchema(s.T(), "publishing", migrations.FS)
	s.store = postgres_snapshot_store.New(s.db)
	s.rows = rows_query.NewPostgresQuery(s.db)
	s.summaries = snapshots_query.NewPostgresQuery(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

var monday = time.Date(2026, 10, 26, 8, 17, 7, 0, time.UTC)

func row(takenAt time.Time, post string, analytics string) stats.Row {
	return stats.Row{
		Version: stats.RowVersion, TakenAt: takenAt, PostID: stats.PostID(post), Platform: stats.TikTok,
		Analytics: json.RawMessage(analytics), Metadata: json.RawMessage(`{"teaserBattle":"entry-7"}`),
	}
}

func (s *testSuite) keep(takenAt time.Time, posts int, rows ...stats.Row) {
	s.T().Helper()
	s.Require().NoError(s.store.Keep(s.T().Context(), stats.Report{TakenAt: takenAt, Since: takenAt.AddDate(0, 0, -28), Posts: posts}, rows))
}

func (s *testSuite) TestARowKeptIsReadBackWithItsNumbersAsTheyCame() {
	s.keep(monday, 1, row(monday, "post-1", `{"completionRate":0.310,"aNewMetric":{"x":1}}`))

	listed, err := s.rows.Rows(s.T().Context(), time.Time{}, time.Time{}, 0, "")

	s.Require().NoError(err)
	s.Require().Len(listed.GetRows(), 1)
	kept := listed.GetRows()[0]
	s.Equal(monday, kept.GetTakenAt().AsTime())
	s.Equal("post-1", kept.GetPostId())
	s.Equal("tiktok", kept.GetPlatform())
	s.Contains(kept.GetDocument(), `"completionRate": 0.310`, "a number as it was written")
	s.Contains(kept.GetDocument(), `"teaserBattle": "entry-7"`, "the post's metadata, whatever posted it")
	s.Empty(listed.GetNextPage())
}

func (s *testSuite) TestASnapshotKeptAgainIsKeptOnceWithItsLastWords() {
	s.keep(monday, 1, row(monday, "post-1", `{"views":1}`))
	s.keep(monday, 2, row(monday, "post-1", `{"views":2}`), row(monday, "post-2", `{"views":5}`))

	listed, err := s.rows.Rows(s.T().Context(), time.Time{}, time.Time{}, 0, "")
	s.Require().NoError(err)
	s.Require().Len(listed.GetRows(), 2)
	s.Contains(listed.GetRows()[0].GetDocument(), `"views": 2`)

	summaries, err := s.summaries.Snapshots(s.T().Context(), time.Time{}, time.Time{}, 0)
	s.Require().NoError(err)
	s.Require().Len(summaries.GetSnapshots(), 1)
	s.Equal(uint32(2), summaries.GetSnapshots()[0].GetPosts())
}

func (s *testSuite) TestTheNewestSnapshotIsTheOneTakenLast() {
	latest, err := s.store.Latest(s.T().Context())
	s.Require().NoError(err)
	s.True(latest.IsZero(), "none kept")

	s.keep(monday, 0)
	s.keep(monday.Add(24*time.Hour), 0)

	latest, err = s.store.Latest(s.T().Context())
	s.Require().NoError(err)
	s.Equal(monday.Add(24*time.Hour), latest)
}

func (s *testSuite) TestTheRowsOfAWindowComeAPageAtATimeOldestFirst() {
	for day := range 3 {
		takenAt := monday.Add(time.Duration(day) * 24 * time.Hour)
		s.keep(takenAt, 2, row(takenAt, "post-a", `{}`), row(takenAt, "post-b", `{}`))
	}

	first, err := s.rows.Rows(s.T().Context(), monday.Add(time.Hour), time.Time{}, 3, "")
	s.Require().NoError(err)
	second, err := s.rows.Rows(s.T().Context(), monday.Add(time.Hour), time.Time{}, 3, first.GetNextPage())
	s.Require().NoError(err)

	s.Require().Len(first.GetRows(), 3, "monday is before the window")
	s.Equal(monday.Add(24*time.Hour), first.GetRows()[0].GetTakenAt().AsTime())
	s.NotEmpty(first.GetNextPage())
	s.Require().Len(second.GetRows(), 1)
	s.Equal("post-b", second.GetRows()[0].GetPostId())
	s.Empty(second.GetNextPage())
}

func (s *testSuite) TestAPageThisQueryDidNotGiveIsRefused() {
	for _, page := range []string{"not base64!", "e30"} {
		_, err := s.rows.Rows(s.T().Context(), time.Time{}, time.Time{}, 0, page)

		s.ErrorIs(err, rows_query.ErrInvalidPage, page)
	}
}

func (s *testSuite) TestTheSummariesComeNewestFirstWithEachTikTokLane() {
	s.Require().NoError(s.store.Keep(s.T().Context(), stats.Report{
		TakenAt: monday, Posts: 3, TikTokAccounts: []stats.TikTokLane{
			{TikTokAccount: stats.TikTokAccount{ID: "tt-1", Username: "clickplanet", Type: "business", CommercialMusic: true}},
		},
		Unread: []stats.Unread{{PostID: "post-9", Reason: "zernio answered 500"}},
	}, nil))
	s.keep(monday.Add(24*time.Hour), 4)

	listed, err := s.summaries.Snapshots(s.T().Context(), time.Time{}, time.Time{}, 0)

	s.Require().NoError(err)
	s.Require().Len(listed.GetSnapshots(), 2)
	s.Equal(uint32(4), listed.GetSnapshots()[0].GetPosts())
	older := listed.GetSnapshots()[1]
	s.Equal("clickplanet", older.GetTiktokAccounts()[0].GetUsername())
	s.True(older.GetTiktokAccounts()[0].GetLive(), "the music library answered")
	s.Equal("zernio answered 500", older.GetUnread()[0].GetReason())
}

func (s *testSuite) TestAPostsLinkIsTheNewestAddressASnapshotRead() {
	at := func(takenAt time.Time, platform, url string) stats.Row {
		read := row(takenAt, "post-1", `{}`)
		read.Platform, read.URL = platform, url
		return read
	}
	s.keep(monday, 1, at(monday, stats.TikTok, "https://tiktok.example/old"))
	tuesday := monday.Add(24 * time.Hour)
	s.keep(tuesday, 1, at(tuesday, stats.TikTok, "https://tiktok.example/7"), at(tuesday, stats.YouTube, ""))

	links, err := links_query.NewPostgresQuery(s.db).Links(s.T().Context(), []string{"post-1", "post-2"})

	s.Require().NoError(err)
	s.Require().Len(links.GetLinks(), 1, "no address is no link")
	s.Equal("post-1", links.GetLinks()[0].GetPostId())
	s.Equal("tiktok", links.GetLinks()[0].GetPlatform())
	s.Equal("https://tiktok.example/7", links.GetLinks()[0].GetUrl())

	none, err := links_query.NewPostgresQuery(s.db).Links(s.T().Context(), nil)
	s.Require().NoError(err)
	s.Empty(none.GetLinks())
}

// The clips module's tables, as they were until 2026-11-10, with a snapshot in them.
var clipsTables = fstest.MapFS{"20261025090000_snapshots.up.sql": {Data: []byte(`
	CREATE TABLE snapshots (taken_at timestamptz PRIMARY KEY, summary jsonb NOT NULL);
	CREATE TABLE snapshot_rows (
		taken_at timestamptz NOT NULL REFERENCES snapshots (taken_at) ON DELETE CASCADE,
		post_id text NOT NULL, platform text NOT NULL, document jsonb NOT NULL,
		PRIMARY KEY (taken_at, post_id, platform)
	);
	INSERT INTO snapshots VALUES ('2026-10-26T05:41:07Z', '{"takenAt":"2026-10-26T05:41:07Z","posts":7}');
	INSERT INTO snapshot_rows VALUES ('2026-10-26T05:41:07Z', 'post-1', 'tiktok', '{"version":1,"metadata":{"storyFlag":"fr"}}');
`)}}

func (s *testSuite) TestTheClipsModulesSnapshotsAreCopiedOnceAndLeftThere() {
	server := cppg.StartTestServer(s.T())
	clips := server.OpenSchema(s.T(), "clips", clipsTables)
	publishing := server.OpenSchema(s.T(), "publishing", migrations.FS)

	summaries, err := snapshots_query.NewPostgresQuery(publishing).Snapshots(s.T().Context(), time.Time{}, time.Time{}, 0)
	s.Require().NoError(err)
	s.Require().Len(summaries.GetSnapshots(), 1)
	s.Equal(uint32(7), summaries.GetSnapshots()[0].GetPosts())
	rows, err := rows_query.NewPostgresQuery(publishing).Rows(s.T().Context(), time.Time{}, time.Time{}, 0, "")
	s.Require().NoError(err)
	s.Require().Len(rows.GetRows(), 1)
	s.Contains(rows.GetRows()[0].GetDocument(), "storyFlag")

	var left int
	s.Require().NoError(clips.On(s.T().Context()).QueryRowContext(s.T().Context(), `SELECT count(*) FROM snapshot_rows`).Scan(&left))
	s.Equal(1, left, "the clips tables are left as they were")
}

package postgres_snapshot_store_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_rows_handler/rows_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_snapshots_handler/snapshots_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats/snapshots"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats/snapshots/postgres_snapshot_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

// The store keeps, and the two queries read what it kept: one suite, one container.
func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db        *cppg.Postgres
	store     *postgres_snapshot_store.Store
	rows      *rows_query.PostgresQuery
	summaries *snapshots_query.PostgresQuery
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "clips", migrations.FS)
	s.store = postgres_snapshot_store.New(s.db)
	s.rows = rows_query.NewPostgresQuery(s.db)
	s.summaries = snapshots_query.NewPostgresQuery(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

var monday = time.Date(2026, 10, 26, 5, 41, 7, 0, time.UTC)

func (s *testSuite) keep(takenAt time.Time, summary string, documents map[string]string) {
	s.T().Helper()

	rows := make([]snapshots.Row, 0, len(documents))
	times := make([]time.Time, 0, len(documents))
	for post, document := range documents {
		rows = append(rows, snapshots.Row{PostID: post, Platform: "tiktok", Document: json.RawMessage(document)})
		times = append(times, takenAt)
	}
	part, err := snapshots.NewPart(takenAt, json.RawMessage(summary), rows, times)
	s.Require().NoError(err)
	s.Require().NoError(s.store.Keep(s.T().Context(), part))
}

func (s *testSuite) TestARowKeptIsReadBackWithItsNumbersAsTheyCame() {
	s.keep(monday, `{"takenAt":"2026-10-26T05:41:07Z","posts":1,"rows":1}`,
		map[string]string{"post-1": `{"version":1,"analytics":{"completionRate":0.310,"aNewMetric":{"x":1}}}`})

	listed, err := s.rows.Rows(s.T().Context(), time.Time{}, time.Time{}, 0, "")

	s.Require().NoError(err)
	s.Require().Len(listed.GetRows(), 1)
	row := listed.GetRows()[0]
	s.Equal(monday, row.GetTakenAt().AsTime())
	s.Equal("post-1", row.GetPostId())
	s.Equal("tiktok", row.GetPlatform())
	s.JSONEq(`{"version":1,"analytics":{"completionRate":0.310,"aNewMetric":{"x":1}}}`, row.GetDocument())
	s.Contains(row.GetDocument(), "0.310", "a number as it was written")
	s.Empty(listed.GetNextPage())
}

func (s *testSuite) TestAPartSentAgainIsKeptOnceWithItsLastWords() {
	s.keep(monday, `{"posts":1}`, map[string]string{"post-1": `{"views":1}`})
	s.keep(monday, `{"posts":2}`, map[string]string{"post-1": `{"views":2}`, "post-2": `{"views":5}`})

	listed, err := s.rows.Rows(s.T().Context(), time.Time{}, time.Time{}, 0, "")
	s.Require().NoError(err)
	s.Require().Len(listed.GetRows(), 2)
	s.JSONEq(`{"views":2}`, listed.GetRows()[0].GetDocument())

	summaries, err := s.summaries.Snapshots(s.T().Context(), time.Time{}, time.Time{}, 0)
	s.Require().NoError(err)
	s.Require().Len(summaries.GetSnapshots(), 1)
	s.Equal(uint32(2), summaries.GetSnapshots()[0].GetPosts())
}

func (s *testSuite) TestTheRowsOfAWindowComeAPageAtATimeOldestFirst() {
	for day := range 3 {
		takenAt := monday.Add(time.Duration(day) * 24 * time.Hour)
		s.keep(takenAt, `{}`, map[string]string{"post-a": `{}`, "post-b": `{}`})
	}

	first, err := s.rows.Rows(s.T().Context(), monday.Add(time.Hour), time.Time{}, 3, "")
	s.Require().NoError(err)
	second, err := s.rows.Rows(s.T().Context(), monday.Add(time.Hour), time.Time{}, 3, first.GetNextPage())
	s.Require().NoError(err)

	s.Require().Len(first.GetRows(), 3, "monday is before the window")
	s.Equal(monday.Add(24*time.Hour), first.GetRows()[0].GetTakenAt().AsTime())
	s.Equal("post-a", first.GetRows()[0].GetPostId())
	s.NotEmpty(first.GetNextPage())
	s.Require().Len(second.GetRows(), 1)
	s.Equal("post-b", second.GetRows()[0].GetPostId())
	s.Equal(monday.Add(48*time.Hour), second.GetRows()[0].GetTakenAt().AsTime())
	s.Empty(second.GetNextPage())

	until, err := s.rows.Rows(s.T().Context(), time.Time{}, monday.Add(24*time.Hour), 0, "")
	s.Require().NoError(err)
	s.Len(until.GetRows(), 2, "until is not in the window")
}

func (s *testSuite) TestAPageThisQueryDidNotGiveIsRefused() {
	for _, page := range []string{"not base64!", "e30"} {
		_, err := s.rows.Rows(s.T().Context(), time.Time{}, time.Time{}, 0, page)

		s.ErrorIs(err, rows_query.ErrInvalidPage, page)
	}
}

func (s *testSuite) TestTheSummariesComeNewestFirstAndReadAsTheWireWroteThem() {
	summary := &clipsv1.StatsSnapshot{TakenAt: timestamppb.New(monday), Posts: 3, TiktokAccounts: []*clipsv1.TikTokAccount{
		{Username: "clickplanet", AccountType: "business", Live: true},
	}}
	s.keep(monday, `{"takenAt":"2026-10-26T05:41:07Z","posts":3,"tiktokAccounts":[{"username":"clickplanet","accountType":"business","live":true}],"aKeyOfANewerJob":1}`, nil)
	s.keep(monday.Add(24*time.Hour), `{"takenAt":"2026-10-27T05:41:07Z","posts":4}`, nil)

	listed, err := s.summaries.Snapshots(s.T().Context(), time.Time{}, time.Time{}, 0)

	s.Require().NoError(err)
	s.Require().Len(listed.GetSnapshots(), 2)
	s.Equal(uint32(4), listed.GetSnapshots()[0].GetPosts())
	s.Equal(summary.GetTiktokAccounts()[0].GetUsername(), listed.GetSnapshots()[1].GetTiktokAccounts()[0].GetUsername())
	s.True(listed.GetSnapshots()[1].GetTiktokAccounts()[0].GetLive())

	one, err := s.summaries.Snapshots(s.T().Context(), time.Time{}, time.Time{}, 1)
	s.Require().NoError(err)
	s.Len(one.GetSnapshots(), 1)
}

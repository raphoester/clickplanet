package take_snapshot_usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats/usecases/take_snapshot_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now = time.Date(2026, 10, 24, 5, 41, 0, 0, time.UTC)
	out = now.Add(-30 * time.Hour)
)

type source struct {
	accounts   stats.Accounts
	posts      []stats.Post
	answers    map[stats.PostID][]error
	retentions []error

	since    time.Time
	asked    []stats.PostID
	curvesOf []string
}

func (s *source) Accounts(context.Context) (stats.Accounts, error) { return s.accounts, nil }

func (s *source) Posts(_ context.Context, since time.Time) ([]stats.Post, error) {
	s.since = since
	return s.posts, nil
}

func (s *source) Analytics(_ context.Context, id stats.PostID) (stats.Answer, error) {
	s.asked = append(s.asked, id)
	if failures := s.answers[id]; len(failures) > 0 {
		s.answers[id] = failures[1:]
		if failures[0] != nil {
			return stats.Answer{}, failures[0]
		}
	}
	return stats.Answer{HTTPStatus: 200, Entries: []stats.Entry{
		{Platform: "tiktok", Raw: json.RawMessage(`{"platform":"tiktok","analytics":{"views":1}}`)},
	}}, nil
}

func (s *source) Retention(_ context.Context, account, video string, asOf time.Time) (stats.Retention, error) {
	s.curvesOf = append(s.curvesOf, account+"/"+video+"@"+asOf.Format(time.DateOnly))
	if len(s.retentions) > 0 {
		failure := s.retentions[0]
		s.retentions = s.retentions[1:]
		if failure != nil {
			return stats.Retention{}, failure
		}
	}
	return stats.Retention{HTTPStatus: 200, Body: json.RawMessage(`{"retentionCurve":[{}]}`)}, nil
}

type store struct {
	kept   []stats.Row
	report stats.Report
	calls  int
}

func (s *store) Keep(_ context.Context, report stats.Report, rows []stats.Row) error {
	s.kept, s.report = rows, report
	s.calls++
	return nil
}

func postOn(id stats.PostID, published bool) stats.Post {
	at := time.Time{}
	if published {
		at = out
	}
	return stats.Post{ID: id, Metadata: json.RawMessage(`{"storyFlag":"fr"}`), Targets: []stats.Target{
		{Platform: "tiktok", AccountID: "tt-1", PlatformPostID: "7", PublishedAt: at},
		{Platform: "youtube", AccountID: "yt-1", PlatformPostID: "abc-" + string(id), PublishedAt: at},
	}}
}

func snapshot(t *testing.T, from *source, since time.Time) (stats.Report, *store, *posts.RecordedPause, error) {
	t.Helper()

	clock := cptime.NewFixedClock(now)
	pause := posts.NewRecordedPause(clock)
	kept := &store{}
	report, err := take_snapshot_usecase.New(from, kept, clock, since, posts.Backoff{time.Minute, 5 * time.Minute}, pause).Execute(t.Context())
	return report, kept, pause, err
}

func TestWithNoAnalyticsNothingIsReadOrKept(t *testing.T) {
	from := &source{accounts: stats.Accounts{AnalyticsAccess: false}, posts: []stats.Post{postOn("post-1", true)}}

	_, kept, _, err := snapshot(t, from, time.Time{})

	require.ErrorIs(t, err, stats.ErrNoAnalyticsAccess)
	assert.Empty(t, from.asked)
	assert.Zero(t, kept.calls, "no empty snapshot is written")
}

func TestEveryPostOutIsARowForEachPlatformWithYouTubesCurve(t *testing.T) {
	from := &source{
		accounts: stats.Accounts{AnalyticsAccess: true, TikTok: []stats.TikTokAccount{{ID: "tt-1", Type: "business"}}},
		posts:    []stats.Post{postOn("post-1", true), postOn("post-2", false), postOn("post-3", true)},
	}

	report, kept, _, err := snapshot(t, from, time.Time{})

	require.NoError(t, err)
	assert.Equal(t, stats.DefaultSince(now), from.since)
	assert.Equal(t, []stats.PostID{"post-1", "post-3"}, from.asked, "a post not out has nothing to read")
	assert.Equal(t, []string{"yt-1/abc-post-1@2026-10-24", "yt-1/abc-post-3@2026-10-24"}, from.curvesOf)
	require.Len(t, kept.kept, 4)
	assert.Equal(t, report, kept.report, "the store keeps the report beside the rows")
	assert.Equal(t, "youtube", kept.kept[1].Platform)
	assert.True(t, kept.kept[1].Retention.Curve())
	assert.Nil(t, kept.kept[0].Retention)

	assert.Equal(t, 3, report.Posts)
	assert.Equal(t, 1, report.NotOut)
	assert.Equal(t, map[string]int{"tiktok": 2, "youtube": 2}, report.Rows)
	assert.Equal(t, 2, report.Curves)
	assert.Equal(t, 2, report.TikTokAccounts[0].Rows)
	assert.Equal(t, now, report.TakenAt)
	assert.True(t, report.Complete())
}

func TestTheDayAskedForIsWhereTheSnapshotStarts(t *testing.T) {
	from := &source{accounts: stats.Accounts{AnalyticsAccess: true}}
	since := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)

	_, kept, _, err := snapshot(t, from, since)

	require.NoError(t, err)
	assert.Equal(t, since, from.since)
	assert.Equal(t, 1, kept.calls, "a snapshot of nothing is still kept: the day it was taken shows")
}

func TestAPostThatCannotBeReadIsReportedAndTheOthersAreStillKept(t *testing.T) {
	from := &source{
		accounts: stats.Accounts{AnalyticsAccess: true},
		posts:    []stats.Post{postOn("post-1", true), postOn("post-2", true)},
		answers:  map[stats.PostID][]error{"post-1": {errors.New("zernio answered 404 to the analytics of the post post-1: Post not found")}},
	}

	report, kept, _, err := snapshot(t, from, time.Time{})

	require.NoError(t, err)
	assert.Len(t, kept.kept, 2)
	assert.Equal(t, stats.PostID("post-2"), kept.kept[0].PostID)
	assert.Equal(t, []stats.Unread{{PostID: "post-1", Reason: "zernio answered 404 to the analytics of the post post-1: Post not found"}}, report.Unread)
	assert.False(t, report.Complete())
}

func TestAPostWhoseCurveCannotBeReadIsKeptWithoutItAndReported(t *testing.T) {
	from := &source{
		accounts:   stats.Accounts{AnalyticsAccess: true},
		posts:      []stats.Post{postOn("post-1", true)},
		retentions: []error{errors.New("no")},
	}

	report, kept, _, err := snapshot(t, from, time.Time{})

	require.NoError(t, err)
	require.Len(t, kept.kept, 2)
	assert.Nil(t, kept.kept[1].Retention)
	require.Len(t, report.Unread, 1)
	assert.Equal(t, "the retention of abc-post-1: no", report.Unread[0].Reason)
}

func TestTheAddOnTurnedOffMidRunFailsTheRunAndKeepsNothing(t *testing.T) {
	from := &source{
		accounts: stats.Accounts{AnalyticsAccess: true},
		posts:    []stats.Post{postOn("post-1", true), postOn("post-2", true)},
		answers:  map[stats.PostID][]error{"post-2": {stats.ErrNoAnalyticsAccess}},
	}

	_, kept, _, err := snapshot(t, from, time.Time{})

	require.ErrorIs(t, err, stats.ErrNoAnalyticsAccess)
	assert.Zero(t, kept.calls)
}

func TestABusyServiceIsAskedAgainAfterAWait(t *testing.T) {
	from := &source{
		accounts: stats.Accounts{AnalyticsAccess: true},
		posts:    []stats.Post{postOn("post-1", true)},
		answers: map[stats.PostID][]error{"post-1": {
			&posts.BusyError{Reason: "slow down"}, &posts.BusyError{RetryAfter: 2 * time.Minute, Reason: "slow down"},
		}},
	}

	report, kept, pause, err := snapshot(t, from, time.Time{})

	require.NoError(t, err)
	assert.Equal(t, []time.Duration{time.Minute, 5 * time.Minute}, pause.Waited)
	assert.Len(t, from.asked, 3)
	assert.Len(t, kept.kept, 2)
	assert.True(t, report.Complete())
}

func TestAServiceBusyForTooLongLosesThatPostOnly(t *testing.T) {
	busy := &posts.BusyError{Reason: "slow down"}
	from := &source{
		accounts: stats.Accounts{AnalyticsAccess: true},
		posts:    []stats.Post{postOn("post-1", true), postOn("post-2", true)},
		answers:  map[stats.PostID][]error{"post-1": {busy, busy, busy}},
	}

	report, kept, _, err := snapshot(t, from, time.Time{})

	require.NoError(t, err)
	assert.Len(t, kept.kept, 2)
	require.Len(t, report.Unread, 1)
	assert.Equal(t, stats.PostID("post-1"), report.Unread[0].PostID)
}

// YouTube's daily quota spent answers every curve with a wait until midnight Pacific: one is enough to know.
func TestACurveRefusedForLongerThanTheBackoffStopsTheCurvesOfTheRun(t *testing.T) {
	spent := &posts.BusyError{RetryAfter: time.Hour, Reason: "YouTube Analytics daily quota exhausted"}
	from := &source{
		accounts:   stats.Accounts{AnalyticsAccess: true},
		posts:      []stats.Post{postOn("post-1", true), postOn("post-2", true), postOn("post-3", true)},
		retentions: []error{nil, spent},
	}

	report, kept, _, err := snapshot(t, from, time.Time{})

	require.NoError(t, err)
	assert.Equal(t, []string{"yt-1/abc-post-1@2026-10-24", "yt-1/abc-post-2@2026-10-24"}, from.curvesOf, "post-3's is not asked")
	assert.Len(t, kept.kept, 6, "every post's rows are kept, its analytics too")
	assert.True(t, kept.kept[1].Retention.Curve())
	require.Len(t, report.Unread, 2)
	assert.Equal(t, stats.PostID("post-2"), report.Unread[0].PostID)
	assert.Contains(t, report.Unread[0].Reason, "the retention of abc-post-2: the posting service is busy for 1h0m0s: YouTube Analytics daily quota exhausted")
	assert.Equal(t, stats.PostID("post-3"), report.Unread[1].PostID)
	assert.Contains(t, report.Unread[1].Reason, "the retention of abc-post-3: not asked: the posting service is busy for 1h0m0s: YouTube Analytics daily quota exhausted")
}

func TestACurveThatFailsOtherwiseStopsNoOther(t *testing.T) {
	from := &source{
		accounts:   stats.Accounts{AnalyticsAccess: true},
		posts:      []stats.Post{postOn("post-1", true), postOn("post-2", true)},
		retentions: []error{errors.New("no")},
	}

	report, _, _, err := snapshot(t, from, time.Time{})

	require.NoError(t, err)
	assert.Len(t, from.curvesOf, 2)
	assert.Len(t, report.Unread, 1)
}

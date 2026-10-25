package take_snapshot_usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	// Past 2 days, so a curve is due.
	out = now.Add(-50 * time.Hour)
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
	last   []stats.Row
	kept   []stats.Row
	report stats.Report
	calls  int
}

func (s *store) Last(context.Context) ([]stats.Row, error) { return s.last, nil }

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
	return snapshotAfter(t, from, since, nil)
}

// last: the rows of the snapshot kept before.
func snapshotAfter(t *testing.T, from *source, since time.Time, last []stats.Row) (stats.Report, *store, *posts.RecordedPause, error) {
	t.Helper()

	clock := cptime.NewFixedClock(now)
	pause := posts.NewRecordedPause(clock)
	kept := &store{last: last}
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

func TestAPostWhoseCurveCannotBeReadKeepsItsRowWithWhyTheCurveIsMissing(t *testing.T) {
	from := &source{
		accounts:   stats.Accounts{AnalyticsAccess: true},
		posts:      []stats.Post{postOn("post-1", true)},
		retentions: []error{errors.New("no")},
	}

	report, kept, _, err := snapshot(t, from, time.Time{})

	require.NoError(t, err)
	require.Len(t, kept.kept, 2)
	assert.Equal(t, 200, kept.kept[1].Answer.HTTPStatus, "the rest of the row is kept")
	assert.InDelta(t, 50, kept.kept[1].AgeHours, 0.01)
	assert.Nil(t, kept.kept[1].Retention)
	assert.Equal(t, "no", kept.kept[1].RetentionMissing)
	assert.True(t, report.Complete(), "a curve missing loses no post")
	assert.Equal(t, map[string]int{"no": 1}, report.RetentionMissing)
}

func TestOnceYouTubesQuotaIsSpentNoCurveIsAskedAgain(t *testing.T) {
	from := &source{
		accounts:   stats.Accounts{AnalyticsAccess: true},
		posts:      []stats.Post{postOn("post-1", true), postOn("post-2", true), postOn("post-3", true)},
		retentions: []error{nil, fmt.Errorf("%w: daily quota exhausted", stats.ErrRetentionQuota)},
	}

	report, kept, pause, err := snapshot(t, from, time.Time{})

	require.NoError(t, err)
	assert.Len(t, from.curvesOf, 2, "the third video is not asked")
	assert.Empty(t, pause.Waited, "a quota spent is not waited out")
	assert.Equal(t, 2, report.RetentionAsked)
	require.Len(t, kept.kept, 6)
	assert.True(t, kept.kept[1].Retention.Curve())
	assert.Equal(t, stats.QuotaSpent, kept.kept[3].RetentionMissing)
	assert.Equal(t, stats.QuotaSpent, kept.kept[5].RetentionMissing)
	assert.Equal(t, 200, kept.kept[5].Answer.HTTPStatus)
	assert.Equal(t, map[string]int{stats.QuotaSpent: 2}, report.RetentionMissing)
	assert.True(t, report.Complete())
}

func TestAVideoUnderTwoDaysOldIsNotAskedForItsCurve(t *testing.T) {
	young := postOn("post-1", true)
	for i := range young.Targets {
		young.Targets[i].PublishedAt = now.Add(-30 * time.Hour)
	}
	from := &source{accounts: stats.Accounts{AnalyticsAccess: true}, posts: []stats.Post{young}}

	report, kept, _, err := snapshot(t, from, time.Time{})

	require.NoError(t, err)
	assert.Empty(t, from.curvesOf)
	assert.Nil(t, kept.kept[1].Retention)
	assert.Empty(t, kept.kept[1].RetentionMissing, "a curve not due is not missing")
	assert.Zero(t, report.RetentionAsked)
}

func TestACurveReadPastTheAgeItWasDueAtIsCarriedAndNotAskedAgain(t *testing.T) {
	read := now.Add(-time.Hour)
	last := []stats.Row{{
		TakenAt: read, Platform: "youtube", PlatformPostID: "abc-post-1", AgeHours: 49,
		Retention: &stats.Retention{HTTPStatus: 200, Body: json.RawMessage(`{"retentionCurve":[{}]}`)},
	}}
	from := &source{accounts: stats.Accounts{AnalyticsAccess: true}, posts: []stats.Post{postOn("post-1", true)}}

	report, kept, _, err := snapshotAfter(t, from, time.Time{}, last)

	require.NoError(t, err)
	assert.Empty(t, from.curvesOf)
	require.True(t, kept.kept[1].Retention.Curve(), "the curve is carried")
	assert.Equal(t, read, kept.kept[1].Retention.ReadAt, "and says when it was read")
	assert.False(t, kept.kept[1].RetentionRead())
	assert.Equal(t, 1, report.Curves)
}

func TestACurveTheLastSnapshotLacksIsAskedAgain(t *testing.T) {
	last := []stats.Row{{
		TakenAt: now.Add(-24 * time.Hour), Platform: "youtube", PlatformPostID: "abc-post-1", AgeHours: 26,
		RetentionMissing: stats.QuotaSpent,
	}}
	from := &source{accounts: stats.Accounts{AnalyticsAccess: true}, posts: []stats.Post{postOn("post-1", true)}}

	_, kept, _, err := snapshotAfter(t, from, time.Time{}, last)

	require.NoError(t, err)
	assert.Len(t, from.curvesOf, 1)
	assert.True(t, kept.kept[1].RetentionRead())
	assert.InDelta(t, 50, kept.kept[1].Retention.AgeHours, 0.01)
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

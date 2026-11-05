package stats_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats"
)

var (
	takenAt     = time.Date(2026, 10, 24, 5, 41, 0, 0, time.UTC)
	publishedAt = time.Date(2026, 10, 22, 16, 40, 0, 0, time.UTC)
)

const (
	tiktokEntry  = `{"platform":"tiktok","accountId":"tt-1","syncStatus":"synced","analytics":{"views":120,"completionRate":0.31}}` //nolint:gosec // the analytics of a post, not a credential.
	youtubeEntry = `{"platform":"youtube","accountId":"yt-1","syncStatus":"pending","analytics":null}`
	metadata     = `{"storyFlag":"fr","storyContinent":"Europe","storyCountries":["fr","pw"],"record":{"clip":{"variants":{"length":"short"}}}}`
)

func aPost() stats.Post {
	return stats.Post{
		ID: "post-1", Metadata: json.RawMessage(metadata),
		Targets: []stats.Target{
			{Platform: "tiktok", AccountID: "tt-1", PlatformPostID: "7", URL: "https://tiktok.example/7", PublishedAt: publishedAt},
			{Platform: "youtube", AccountID: "yt-1", PlatformPostID: "abc", URL: "https://youtube.example/abc", PublishedAt: publishedAt.Add(-15 * time.Minute)},
			{Platform: "instagram", AccountID: "ig-1"},
		},
	}
}

func anAnswer() stats.Answer {
	return stats.Answer{HTTPStatus: 202, Status: "published", SyncStatus: "pending", Entries: []stats.Entry{
		{Platform: "tiktok", AccountID: "tt-1", Raw: json.RawMessage(tiktokEntry)},
		{Platform: "youtube", AccountID: "yt-1", Raw: json.RawMessage(youtubeEntry)},
	}}
}

func TestAPostIsARowForEachPlatformItIsOutOnWithItsEntryAndItsMetadataAsTheyCame(t *testing.T) {
	retention := &stats.Retention{HTTPStatus: 200, Body: json.RawMessage(`{"retentionCurve":[]}`)}

	rows := stats.RowsOf(aPost(), anAnswer(), map[string]*stats.Retention{"abc": retention}, takenAt)

	require.Len(t, rows, 2, "a platform it is not out on has nothing to measure")
	assert.Equal(t, stats.Row{
		Version: stats.RowVersion, TakenAt: takenAt, PostID: "post-1", Platform: "tiktok", AccountID: "tt-1",
		PlatformPostID: "7", URL: "https://tiktok.example/7", PublishedAt: publishedAt, AgeHours: 37.02,
		Answer: anAnswer(), Analytics: json.RawMessage(tiktokEntry), Metadata: json.RawMessage(metadata),
	}, rows[0])
	assert.JSONEq(t, youtubeEntry, string(rows[1].Analytics))
	assert.Same(t, retention, rows[1].Retention)
	assert.InDelta(t, 37.27, rows[1].AgeHours, 0.001)
}

func TestAPlatformTheAnswerHasNoEntryForIsARowWithNullAnalytics(t *testing.T) {
	rows := stats.RowsOf(aPost(), stats.Answer{HTTPStatus: 424}, nil, takenAt)

	require.Len(t, rows, 2)
	assert.JSONEq(t, `null`, string(rows[0].Analytics))
	assert.Nil(t, rows[0].Retention)
}

func TestAPostOutOnNoPlatformIsNotOut(t *testing.T) {
	post := stats.Post{Targets: []stats.Target{{Platform: "tiktok"}, {Platform: "youtube"}}}

	assert.False(t, post.Out())
	assert.True(t, aPost().Out())
}

func TestBusinessMetricsAreTheOnesOnlyTikTokForBusinessGives(t *testing.T) {
	for entry, business := range map[string]bool{
		`{"analytics":{"views":120,"likes":4,"completionRate":0,"impressionSources":{},"audienceCountries":{}}}`: false,
		`{"analytics":null}`:                    false,
		`null`:                                  false,
		`{"analytics":{"completionRate":0.31}}`: true,
		`{"analytics":{"impressionSources":{"forYou":0.9}}}`: true,
		`{"analytics":{"audienceCountries":{"FR":1}}}`:       true,
		`{"analytics":{"retentionCurve":{"0":1}}}`:           true,
		`{"analytics":{"profileViews":2}}`:                   true,
	} {
		assert.Equal(t, business, stats.Row{Analytics: json.RawMessage(entry)}.BusinessMetrics(), entry)
	}
}

func TestATikTokLaneIsLiveWhenItsLaneAnswersWhateverTheTypeSaid(t *testing.T) {
	accounts := stats.Accounts{AnalyticsAccess: true, TikTok: []stats.TikTokAccount{
		{ID: "tt-1", Username: "clickplanet", Type: "business"},
		{ID: "tt-2", Type: "business", CommercialMusic: true},
		{ID: "tt-3", Type: "personal", CommercialMusic: true},
		{ID: "tt-4", Type: "business"},
	}}
	rows := []stats.Row{
		{Platform: "tiktok", AccountID: "tt-1", Analytics: json.RawMessage(`{"analytics":{"completionRate":0.4}}`)},
		{Platform: "tiktok", AccountID: "tt-1", Analytics: json.RawMessage(`{"syncStatus":"pending","analytics":{"views":3}}`)},
		{
			Platform: "youtube", AccountID: "yt-1", Analytics: json.RawMessage(youtubeEntry),
			Retention: &stats.Retention{HTTPStatus: 412, Body: json.RawMessage(`{"code":"youtube_analytics_scope_missing"}`)},
		},
	}

	report := stats.ReportOf(accounts, rows)

	require.Len(t, report.TikTokAccounts, 4)
	assert.Equal(t, 2, report.TikTokAccounts[0].Rows)
	assert.Equal(t, 1, report.TikTokAccounts[0].BusinessRows)
	assert.True(t, report.TikTokAccounts[0].Live(), "a business metric came")
	assert.True(t, report.TikTokAccounts[1].Live(), "the music library answers")
	assert.True(t, report.TikTokAccounts[2].Live(), "the lane answers, whatever TikTok called the account when it was connected")
	assert.False(t, report.TikTokAccounts[3].Live(), "a business account whose lane gives nothing yet")
	assert.Equal(t, map[string]int{"tiktok": 2, "youtube": 1}, report.Rows)
	assert.Equal(t, 3, report.RowCount())
	assert.Equal(t, 2, report.Syncing)
	assert.Equal(t, map[string]int{"412 youtube_analytics_scope_missing": 1}, report.CurvesRefused)
	assert.True(t, report.Complete())
}

func TestARetentionSaysWhetherItHoldsACurveAndWhyNot(t *testing.T) {
	curve := &stats.Retention{HTTPStatus: 200, Body: json.RawMessage(`{"retentionCurve":[{"audienceWatchRatio":1}]}`)}
	empty := &stats.Retention{HTTPStatus: 200, Body: json.RawMessage(`{"retentionCurve":[]}`)}
	denied := &stats.Retention{HTTPStatus: 403, Body: json.RawMessage(`"Access denied"`)}

	assert.True(t, curve.Curve())
	assert.False(t, empty.Curve())
	assert.Equal(t, "empty", empty.Why())
	assert.Equal(t, "403", denied.Why())
	var none *stats.Retention
	assert.False(t, none.Curve())
}

func TestASnapshotReadsTheLastFourWeeksByDefault(t *testing.T) {
	assert.Equal(t, time.Date(2026, 9, 26, 5, 41, 0, 0, time.UTC), stats.DefaultSince(takenAt))
}

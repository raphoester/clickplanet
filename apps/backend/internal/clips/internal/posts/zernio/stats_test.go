package zernio_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/zernio"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats"
)

// What one call answers, by method and path.
type statsFake map[string]answer

func (f statsFake) serve(t *testing.T) (*zernio.Service, *[]*url.URL) {
	t.Helper()

	var asked []*url.URL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, r.URL)
		assert.Equal(t, "Bearer the-key", r.Header.Get("Authorization"))
		said, known := f[r.Method+" "+r.URL.Path]
		if !known {
			t.Errorf("zernio was asked %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		said.write(w)
	}))
	t.Cleanup(server.Close)

	service, err := zernio.New(zernio.Config{APIKey: "the-key", BaseURL: server.URL}, server.Client())
	require.NoError(t, err)
	return service, &asked
}

func TestTheAccountsSayWhetherAnalyticsAreOnAndHowEachTikTokAccountIsConnected(t *testing.T) {
	service, asked := statsFake{
		"GET /v1/accounts": {status: http.StatusOK, body: `{"hasAnalyticsAccess":true,"accounts":[
			{"_id":"tt-1","platform":"tiktok","isActive":true,"username":"clickplanet","tiktokAccountType":"business"},
			{"_id":"tt-2","platform":"tiktok","isActive":true,"username":"other"},
			{"_id":"tt-3","platform":"tiktok","isActive":false},
			{"_id":"yt-1","platform":"youtube","isActive":true}]}`},
		"GET /v1/accounts/tt-1/tiktok/commercial-music": {status: http.StatusOK, body: `{"tracks":[]}`},
		"GET /v1/accounts/tt-2/tiktok/commercial-music": {status: http.StatusBadRequest, body: `{"error":"Not a business account","code":"tiktok_business_required"}`},
	}.serve(t)

	accounts, err := service.Accounts(t.Context())

	require.NoError(t, err)
	assert.Equal(t, stats.Accounts{AnalyticsAccess: true, TikTok: []stats.TikTokAccount{
		{ID: "tt-1", Username: "clickplanet", Type: "business", CommercialMusic: true},
		{ID: "tt-2", Username: "other", CommercialMusicRefusal: "zernio answered 400 to the commercial music library: Not a business account"},
	}}, accounts)
	assert.Equal(t, "excludeHidden=true", (*asked)[0].RawQuery)
}

func TestAnAccountWithNoAnalyticsAddOnSaysSo(t *testing.T) {
	service, _ := statsFake{
		"GET /v1/accounts": {status: http.StatusOK, body: `{"hasAnalyticsAccess":false,"accounts":[]}`},
	}.serve(t)

	accounts, err := service.Accounts(t.Context())

	require.NoError(t, err)
	assert.False(t, accounts.AnalyticsAccess)
}

func TestThePostsOfTheJobAreTheKeyedOnesOfEveryPageWithTheirTargetsAndTheirMetadata(t *testing.T) {
	keyed := func(id string) string {
		return `{"_id":"` + id + `","status":"published","metadata":{"storyFlag":"fr","storyContinent":"Europe",` +
			`"record":{"schema":"clips.v1.PostRecord","post":{"runSize":6}}},"platforms":[` +
			`{"platform":"tiktok","accountId":"tt-1","status":"published","platformPostId":"7","platformPostUrl":"https://tiktok.example/7","publishedAt":"2026-10-13T16:40:00Z"},` +
			`{"platform":"youtube","accountId":{"_id":"yt-1","platform":"youtube","isActive":true,"profileId":"p"},"status":"failed"}]}`
	}
	// A full page, then the rest.
	first := make([]string, 0, 500)
	first = append(first, keyed("post-1"), `{"_id":"by-hand","status":"published"}`)
	for len(first) < 500 {
		first = append(first, keyed(fmt.Sprintf("post-%d", len(first))))
	}
	pages := []string{
		`{"posts":[` + strings.Join(first, ",") + `],"pagination":{"page":1,"pages":2}}`,
		`{"posts":[` + keyed("post-last") + `],"pagination":{"page":2,"pages":2}}`,
	}
	var asked []url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, r.URL.Query())
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(pages[len(asked)-1]))
	}))
	t.Cleanup(server.Close)
	service, err := zernio.New(zernio.Config{APIKey: "the-key", BaseURL: server.URL}, server.Client())
	require.NoError(t, err)

	read, err := service.Posts(t.Context(), time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC))

	require.NoError(t, err)
	require.Len(t, asked, 2)
	assert.Equal(t, url.Values{"fromDate": {"2026-10-13"}, "limit": {"500"}, "page": {"1"}}, asked[0])
	assert.Equal(t, "2", asked[1].Get("page"))
	require.Len(t, read, 500, "the post made by hand is left out")
	assert.Equal(t, stats.PostID("post-1"), read[0].ID)
	assert.Equal(t, stats.PostID("post-last"), read[499].ID)
	assert.Equal(t, []stats.Target{
		{Platform: "tiktok", AccountID: "tt-1", PlatformPostID: "7", URL: "https://tiktok.example/7", PublishedAt: time.Date(2026, 10, 13, 16, 40, 0, 0, time.UTC)},
		{Platform: "youtube", AccountID: "yt-1"},
	}, read[0].Targets)
	assert.JSONEq(t, `{"storyFlag":"fr","storyContinent":"Europe","record":{"schema":"clips.v1.PostRecord","post":{"runSize":6}}}`, string(read[0].Metadata))
}

const analyticsBody = `{"postId":"post-1","status":"published","syncStatus":"synced","message":null,"platformAnalytics":[
	{"platform":"tiktok","accountId":"tt-1","syncStatus":"synced","analytics":{"views":120,"completionRate":0.31,"aNewMetric":7}},
	{"platform":"youtube","accountId":"yt-1","syncStatus":"pending","analytics":null}]}`

func TestTheAnalyticsOfAPostKeepEachPlatformsEntryAsItCame(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusAccepted, http.StatusFailedDependency} {
		service, asked := statsFake{"GET /v1/analytics": {status: status, body: analyticsBody}}.serve(t)

		answer, err := service.Analytics(t.Context(), "post-1")

		require.NoError(t, err, status)
		assert.Equal(t, "post-1", (*asked)[0].Query().Get("postId"))
		assert.Equal(t, status, answer.HTTPStatus)
		assert.Equal(t, "published", answer.Status)
		assert.Equal(t, "synced", answer.SyncStatus)
		require.Len(t, answer.Entries, 2)
		assert.Equal(t, "tiktok", answer.Entries[0].Platform)
		assert.Equal(t, "tt-1", answer.Entries[0].AccountID)
		assert.JSONEq(t, `{"platform":"tiktok","accountId":"tt-1","syncStatus":"synced","analytics":{"views":120,"completionRate":0.31,"aNewMetric":7}}`,
			string(answer.Entries[0].Raw), "a metric zernio adds later is kept too")
	}
}

func TestAnalyticsWithNoAddOnAreTheErrorThatSaysSo(t *testing.T) {
	service, _ := statsFake{"GET /v1/analytics": {status: http.StatusPaymentRequired, body: `{"error":"Analytics add-on required","code":"analytics_addon_required"}`}}.serve(t)

	_, err := service.Analytics(t.Context(), "post-1")

	require.ErrorIs(t, err, stats.ErrNoAnalyticsAccess)
}

func TestARateLimitedReadIsBusyAndAnotherRefusalIsNot(t *testing.T) {
	busy, _ := statsFake{"GET /v1/analytics": {status: http.StatusTooManyRequests, body: `{"error":"slow down"}`, headers: map[string]string{"Retry-After": "30"}}}.serve(t)
	gone, _ := statsFake{"GET /v1/analytics": {status: http.StatusNotFound, body: `{"error":"Post not found"}`}}.serve(t)

	_, busyErr := busy.Analytics(t.Context(), "post-1")
	_, goneErr := gone.Analytics(t.Context(), "post-1")

	var waiting *posts.BusyError
	require.ErrorAs(t, busyErr, &waiting)
	assert.Equal(t, 30*time.Second, waiting.RetryAfter)
	require.EqualError(t, goneErr, "zernio answered 404 to the analytics of the post post-1: Post not found")
	assert.NotErrorAs(t, goneErr, &waiting)
}

func TestARetentionCurveIsKeptAsItCameAndSoIsWhyThereIsNone(t *testing.T) {
	curve := `{"success":true,"videoId":"abc","durationSeconds":14,"retentionCurve":[{"elapsedVideoTimeRatio":0.01,"audienceWatchRatio":1.2,"relativeRetentionPerformance":0.7}]}`
	scope := `{"success":false,"code":"youtube_analytics_scope_missing","error":"reconnect"}`
	for status, body := range map[int]string{http.StatusOK: curve, http.StatusPreconditionFailed: scope} {
		service, asked := statsFake{"GET /v1/analytics/youtube/video-retention": {status: status, body: body}}.serve(t)

		retention, err := service.Retention(t.Context(), "yt-1", "abc", time.Date(2026, 10, 24, 5, 0, 0, 0, time.UTC))

		require.NoError(t, err)
		assert.Equal(t, url.Values{"videoId": {"abc"}, "accountId": {"yt-1"}, "toDate": {"2026-10-24"}}, (*asked)[0].Query())
		assert.Equal(t, status, retention.HTTPStatus)
		assert.JSONEq(t, body, string(retention.Body))
	}
}

func TestARetentionTheServiceCouldNotAnswerIsBusy(t *testing.T) {
	service, _ := statsFake{"GET /v1/analytics/youtube/video-retention": {status: http.StatusBadGateway, body: `<html>`}}.serve(t)

	_, err := service.Retention(t.Context(), "yt-1", "abc", time.Now())

	var waiting *posts.BusyError
	require.ErrorAs(t, err, &waiting)
}

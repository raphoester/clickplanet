package zernio_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts/zernio"
)

const (
	oneAccountEach = `{"accounts":[{"_id":"acc-1","platform":"tiktok","isActive":true},{"_id":"acc-2","platform":"youtube","isActive":true}]}`
	onBoth         = `"platforms":[{"platform":"tiktok","status":"pending"},{"platform":"youtube","status":"pending"}]`
	kept           = `{"post":{"_id":"post-1","status":"scheduled","scheduledFor":"2026-10-12T16:40:00Z",` + onBoth + `},` +
		`"message":"Post scheduled successfully"}`
	videoKey = "9b2a1d3c"
)

var (
	at   = time.Date(2026, 10, 12, 16, 40, 0, 0, time.UTC)
	both = []calendar.Platform{calendar.TikTok, calendar.YouTube}
)

type zernioFake struct {
	accounts answer
	presign  answer
	posts    answer
	listed   answer

	calls     []string
	queries   map[string]url.Values
	headers   map[string]http.Header
	sent      map[string]any
	presigned map[string]any
}

type answer struct {
	status  int
	body    string
	headers map[string]string
}

func (a answer) write(w http.ResponseWriter) {
	for name, value := range a.headers {
		w.Header().Set(name, value)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(a.status)
	_, _ = fmt.Fprint(w, a.body)
}

func newFake() *zernioFake {
	return &zernioFake{
		accounts: answer{status: http.StatusOK, body: oneAccountEach},
		presign: answer{status: http.StatusOK, body: `{"uploadUrl":"https://storage.example/temp/clip-1.mp4?signature=the-signature",` +
			`"publicUrl":"https://media.example/clip-1.mp4","key":"temp/clip-1.mp4","expiresIn":3600}`},
		posts:   answer{status: http.StatusCreated, body: kept},
		listed:  answer{status: http.StatusOK, body: `{"posts":[]}`},
		headers: map[string]http.Header{},
		queries: map[string]url.Values{},
	}
}

func (f *zernioFake) service(t *testing.T) *zernio.Service {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := r.Method + " " + r.URL.Path
		f.calls = append(f.calls, call)
		f.headers[call] = r.Header.Clone()
		f.queries[call] = r.URL.Query()
		raw, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		switch call {
		case "GET /v1/accounts":
			assert.Equal(t, url.Values{"excludeHidden": {"true"}}, r.URL.Query())
			f.accounts.write(w)
		case "POST /v1/media/presign":
			assert.NoError(t, json.Unmarshal(raw, &f.presigned))
			f.presign.write(w)
		case "GET /v1/posts":
			f.listed.write(w)
		case "POST /v1/posts":
			assert.NoError(t, json.Unmarshal(raw, &f.sent))
			f.posts.write(w)
		default:
			t.Errorf("zernio was asked %s", call)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	service, err := zernio.New(zernio.Config{APIKey: "the-key", BaseURL: server.URL}, server.Client())
	require.NoError(t, err)
	return service
}

func post(t *testing.T, platforms ...calendar.Platform) posts.Post {
	t.Helper()

	if len(platforms) == 0 {
		platforms = both
	}
	post, err := posts.NewPost(posts.Draft{
		Slot: calendar.Slot{At: at, Platforms: platforms}, Media: "https://media.example/clip-1.mp4", Key: videoKey,
		Title:  "FRANCE IS ATTACKING EUROPE",
		TikTok: &posts.TikTokWords{Caption: "FRANCE TAKES OVER EUROPE | CLICKPLANET\nPlay for free 👉 https://clickplanet.lol\n#clickplanet"},
		YouTube: &posts.YouTubeWords{
			Title: "FRANCE TAKES OVER EUROPE | CLICKPLANET", Description: "Play for free 👉 https://clickplanet.lol/?s=yt\n#clickplanet",
			Tags: []string{"clickplanet", "europe"},
		},
		Metadata: map[string]any{"storyFlag": "fr", "record": map[string]any{"schema": "clips.v1.PostRecord"}},
	})
	require.NoError(t, err)
	return post
}

func tiktokTarget() map[string]any {
	return map[string]any{
		"platform": "tiktok", "accountId": "acc-1",
		"customContent": "FRANCE TAKES OVER EUROPE | CLICKPLANET\nPlay for free 👉 https://clickplanet.lol\n#clickplanet",
		"platformSpecificData": map[string]any{
			"privacyLevel": "PUBLIC_TO_EVERYONE", "allowComment": true, "allowDuet": true, "allowStitch": true,
			"contentPreviewConfirmed": true, "expressConsentGiven": true,
		},
	}
}

func TestAVideoIsUploadedThroughALinkThatCarriesNoKey(t *testing.T) {
	fake := newFake()

	upload, err := fake.service(t).MediaUpload(t.Context(), posts.Media{FileName: "clip-1.mp4", ContentType: posts.VideoMP4, Size: 9})

	require.NoError(t, err)
	assert.Equal(t, posts.MediaUpload{
		UploadURL: "https://storage.example/temp/clip-1.mp4?signature=the-signature", Media: "https://media.example/clip-1.mp4",
	}, upload)
	assert.Equal(t, map[string]any{"filename": "clip-1.mp4", "contentType": "video/mp4", "size": float64(9)}, fake.presigned)
	assert.Equal(t, "Bearer the-key", fake.headers["POST /v1/media/presign"].Get("Authorization"))
}

func TestAnUploadZernioRefusesSaysWhy(t *testing.T) {
	fake := newFake()
	fake.presign = answer{status: http.StatusBadRequest, body: `{"error":"File too large"}`}

	_, err := fake.service(t).MediaUpload(t.Context(), posts.Media{FileName: "clip-1.mp4", ContentType: posts.VideoMP4, Size: 9})

	require.ErrorContains(t, err, "zernio answered 400 to the address to upload the video to: File too large")
}

func TestAPostIsOnePostWithATargetOnEachPlatformAndTheMetadataAsTheJobGaveIt(t *testing.T) {
	fake := newFake()

	scheduled, err := fake.service(t).Schedule(t.Context(), post(t))

	require.NoError(t, err)
	assert.Equal(t, posts.Scheduled{ID: "post-1", Refused: map[calendar.Platform]error{}}, scheduled)
	assert.Equal(t, []string{"GET /v1/accounts", "POST /v1/posts"}, fake.calls)
	assert.Equal(t, videoKey, fake.headers["POST /v1/posts"].Get("Idempotency-Key"))
	assert.Equal(t, map[string]any{
		"title":      "FRANCE IS ATTACKING EUROPE",
		"mediaItems": []any{map[string]any{"type": "video", "url": "https://media.example/clip-1.mp4"}},
		"platforms": []any{
			tiktokTarget(),
			map[string]any{
				"platform": "youtube", "accountId": "acc-2", "customContent": "Play for free 👉 https://clickplanet.lol/?s=yt\n#clickplanet",
				"platformSpecificData": map[string]any{
					"title": "FRANCE TAKES OVER EUROPE | CLICKPLANET", "visibility": "public", "madeForKids": false, "categoryId": "20",
				},
			},
		},
		"tags":         []any{"clickplanet", "europe"},
		"scheduledFor": "2026-10-12T16:40:00Z",
		"metadata":     map[string]any{"storyFlag": "fr", "record": map[string]any{"schema": "clips.v1.PostRecord"}},
	}, fake.sent)
}

func TestAPostForTikTokAloneHasNoYouTubeTargetAndNoTags(t *testing.T) {
	fake := newFake()
	fake.posts = answer{status: http.StatusCreated, body: `{"post":{"_id":"post-1","status":"scheduled","platforms":[{"platform":"tiktok"}]}}`}

	scheduled, err := fake.service(t).Schedule(t.Context(), post(t, calendar.TikTok))

	require.NoError(t, err)
	assert.Equal(t, posts.Scheduled{ID: "post-1", Refused: map[calendar.Platform]error{}}, scheduled)
	assert.Equal(t, []any{tiktokTarget()}, fake.sent["platforms"])
	assert.NotContains(t, fake.sent, "tags")
}

func TestTheSamePostAnsweredAgainWithFieldsAddedLaterIsStillRead(t *testing.T) {
	fake := newFake()
	fake.posts = answer{status: http.StatusOK, body: `{"post":{"_id":"post-1","status":"scheduled","addedLater":true,` + onBoth + `},"addedLater":1}`}

	scheduled, err := fake.service(t).Schedule(t.Context(), post(t))

	require.NoError(t, err)
	assert.Equal(t, posts.PostID("post-1"), scheduled.ID)
}

func TestAPlatformThePostZernioKeptIsNotOnIsRefused(t *testing.T) {
	fake := newFake()
	fake.posts = answer{status: http.StatusCreated, body: `{"post":{"_id":"post-1","status":"scheduled","platforms":[{"platform":"tiktok"}]}}`}

	scheduled, err := fake.service(t).Schedule(t.Context(), post(t))

	require.NoError(t, err)
	require.ErrorContains(t, scheduled.Refused[calendar.YouTube], "zernio kept the post without it")
}

func TestAPostZernioDidNotKeepAsScheduledIsAFailure(t *testing.T) {
	fake := newFake()
	fake.posts = answer{status: http.StatusCreated, body: `{"post":{"_id":"post-1","status":"published",` + onBoth + `}}`}

	_, err := fake.service(t).Schedule(t.Context(), post(t))

	require.ErrorContains(t, err, `zernio kept the post as "published", not "scheduled"`)
}

func TestABusyZernioSaysSoWithHowLongToWait(t *testing.T) {
	for name, refused := range map[string]answer{
		"too many requests": {status: http.StatusTooManyRequests, body: `{"error":"Rate limit exceeded"}`, headers: map[string]string{"Retry-After": "90"}},
		"a server error":    {status: http.StatusBadGateway, body: `not json`, headers: map[string]string{"Retry-After": "90"}},
		"the first try still in flight": {
			status: http.StatusConflict, body: `{"error":"in progress","code":"idempotency_conflict"}`, headers: map[string]string{"Retry-After": "90"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			fake := newFake()
			fake.posts = refused

			_, err := fake.service(t).Schedule(t.Context(), post(t))

			var busy *posts.BusyError
			require.ErrorAs(t, err, &busy)
			assert.Equal(t, 90*time.Second, busy.RetryAfter)
		})
	}
}

func TestARefusalIsAnErrorThatSaysWhyAndIsNotBusy(t *testing.T) {
	for name, refused := range map[string]answer{
		"a bad request":     {status: http.StatusBadRequest, body: `{"error":"Selected privacy level is not available for this creator."}`},
		"a wrong key":       {status: http.StatusUnauthorized, body: `{"error":"Invalid API key"}`},
		"the same content":  {status: http.StatusConflict, body: `{"error":"This exact content is already scheduled"}`},
		"a lost connection": {status: http.StatusForbidden, body: `{"error":"Account is disconnected","code":"ACCOUNT_DISCONNECTED"}`},
	} {
		t.Run(name, func(t *testing.T) {
			fake := newFake()
			fake.posts = refused

			_, err := fake.service(t).Schedule(t.Context(), post(t))

			var busy *posts.BusyError
			require.Error(t, err)
			assert.NotErrorAs(t, err, &busy)
			assert.ErrorContains(t, err, fmt.Sprintf("zernio answered %d to the post: ", refused.status))
			assert.NotContains(t, err.Error(), "the-key")
		})
	}
}

func TestAPlatformWithNoAccountOrTwoIsRefusedAndThePostStillGoesToTheOthers(t *testing.T) {
	tiktok := `{"_id":"acc-1","platform":"tiktok","isActive":true}`
	for name, youtube := range map[string]string{
		"none":         ``,
		"disconnected": `,{"_id":"acc-2","platform":"youtube","isActive":false}`,
		"two":          `,{"_id":"acc-2","platform":"youtube","isActive":true},{"_id":"acc-3","platform":"youtube","isActive":true}`,
	} {
		t.Run(name, func(t *testing.T) {
			fake := newFake()
			fake.accounts = answer{status: http.StatusOK, body: `{"accounts":[` + tiktok + youtube + `]}`}
			fake.posts = answer{status: http.StatusCreated, body: `{"post":{"_id":"post-1","status":"scheduled","platforms":[{"platform":"tiktok"}]}}`}

			scheduled, err := fake.service(t).Schedule(t.Context(), post(t))

			require.NoError(t, err)
			assert.Equal(t, posts.PostID("post-1"), scheduled.ID)
			require.Len(t, scheduled.Refused, 1)
			require.ErrorContains(t, scheduled.Refused[calendar.YouTube], "connected youtube accounts, and the game posts to exactly one")
			assert.Equal(t, []any{tiktokTarget()}, fake.sent["platforms"])
			assert.NotContains(t, fake.sent, "tags")
		})
	}
}

func TestNoAccountOnAnyPlatformIsNoPost(t *testing.T) {
	fake := newFake()
	fake.accounts = answer{status: http.StatusOK, body: `{"accounts":[]}`}

	scheduled, err := fake.service(t).Schedule(t.Context(), post(t))

	require.NoError(t, err)
	assert.Empty(t, scheduled.ID)
	assert.Len(t, scheduled.Refused, 2)
	assert.Equal(t, []string{"GET /v1/accounts"}, fake.calls)
}

func TestAListOfAccountsZernioRefusesIsAnError(t *testing.T) {
	fake := newFake()
	fake.accounts = answer{status: http.StatusUnauthorized, body: `{"error":"Invalid API key"}`}

	_, err := fake.service(t).Schedule(t.Context(), post(t))

	require.ErrorContains(t, err, "zernio answered 401 to the list of accounts: Invalid API key")
}

func TestAZernioThatCannotBeReachedIsBusyAndNeverSaysItsAddress(t *testing.T) {
	service, err := zernio.New(zernio.Config{APIKey: "the-key", BaseURL: "http://127.0.0.1:1/api?signature=the-signature"}, http.DefaultClient)
	require.NoError(t, err)

	_, err = service.Schedule(t.Context(), post(t))

	var busy *posts.BusyError
	require.ErrorAs(t, err, &busy)
	assert.NotContains(t, err.Error(), "the-signature")
	assert.NotContains(t, err.Error(), "/api")
}

func TestAnEmptyKeyIsRefusedWhenTheServiceIsBuilt(t *testing.T) {
	_, err := zernio.New(zernio.Config{}, http.DefaultClient)

	require.ErrorContains(t, err, "the zernio api key is empty")
}

const listed = `{"posts":[` +
	`{"_id":"p-1","status":"scheduled","scheduledFor":"2026-10-24T15:00:00Z","metadata":{"storyFlag":"dz","storyContinent":"Africa"},` +
	`"platforms":[{"platform":"tiktok","status":"pending"},{"platform":"youtube","status":"pending"}]},` +
	`{"_id":"p-2","status":"published","scheduledFor":"2026-10-24T09:00:00Z","content":"a post made by hand",` +
	`"platforms":[{"platform":"tiktok","status":"published"},{"platform":"instagram","status":"published"}]},` +
	// Over TikTok's cap: Zernio holds its TikTok target until the oldest post leaves the window.
	`{"_id":"p-3","status":"partial","scheduledFor":"2026-10-24T12:00:00Z",` +
	`"platforms":[{"platform":"youtube","status":"published"},{"platform":"tiktok","status":"pending","scheduledFor":"2026-10-24T19:30:00Z"}]},` +
	`{"_id":"p-4","status":"partial","scheduledFor":"2026-10-24T10:00:00Z",` +
	`"platforms":[{"platform":"youtube","status":"published"},{"platform":"tiktok","status":"failed","errorMessage":"TikTok creator has reached the daily posting limit."}]},` +
	`{"_id":"p-5","status":"failed","scheduledFor":"2026-10-24T08:00:00Z","platforms":[{"platform":"tiktok","status":"failed"}]},` +
	`{"_id":"p-6","status":"cancelled","scheduledFor":"2026-10-24T07:00:00Z","platforms":[{"platform":"tiktok","status":"cancelled"}]},` +
	`{"_id":"p-7","title":"A DRAFT","status":"draft","platforms":[{"platform":"tiktok"}],"addedLater":1},` +
	`{"_id":"p-8","status":"published","platforms":[{"platform":"youtube","status":"published","publishedAt":"2026-10-24T11:00:00Z"}]}` +
	`],"pagination":{"page":1}}`

func TestWhatEachPlatformHoldsIsEveryPostOnItByHandOrNotAtThatPlatformsTime(t *testing.T) {
	fake := newFake()
	fake.listed = answer{status: http.StatusOK, body: listed}

	feed, err := fake.service(t).Posted(t.Context(), time.Date(2026, 10, 23, 15, 0, 0, 0, time.UTC))

	require.NoError(t, err)
	assert.Equal(t, calendar.Feed{
		{At: time.Date(2026, 10, 24, 15, 0, 0, 0, time.UTC), Platforms: both},
		{At: time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC), Platforms: []calendar.Platform{calendar.TikTok}},
		{At: time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC), Platforms: []calendar.Platform{calendar.YouTube}},
		{At: time.Date(2026, 10, 24, 19, 30, 0, 0, time.UTC), Platforms: []calendar.Platform{calendar.TikTok}},
		{At: time.Date(2026, 10, 24, 10, 0, 0, 0, time.UTC), Platforms: []calendar.Platform{calendar.YouTube}},
		{At: time.Date(2026, 10, 24, 11, 0, 0, 0, time.UTC), Platforms: []calendar.Platform{calendar.YouTube}},
	}, feed)
	assert.Equal(t, url.Values{"fromDate": {"2026-10-23"}, "limit": {"500"}}, fake.queries["GET /v1/posts"])
	assert.Equal(t, "Bearer the-key", fake.headers["GET /v1/posts"].Get("Authorization"))
}

func TestWhatEachPlatformHoldsCannotBeReadWhenZernioRefusesTheList(t *testing.T) {
	fake := newFake()
	fake.listed = answer{status: http.StatusUnauthorized, body: `{"error":"Invalid API key"}`}

	_, err := fake.service(t).Posted(t.Context(), at)

	require.ErrorContains(t, err, "zernio answered 401 to the list of posts: Invalid API key")
}

func TestEveryPostIsListedAsTheServiceHoldsItWithItsMetadata(t *testing.T) {
	fake := newFake()
	fake.listed = answer{status: http.StatusOK, body: listed}

	listed, err := fake.service(t).Posts(t.Context(), time.Date(2026, 10, 23, 15, 0, 0, 0, time.UTC))

	require.NoError(t, err)
	require.Len(t, listed, 8)
	metadata, err := structpb.NewStruct(map[string]any{"storyFlag": "dz", "storyContinent": "Africa"})
	require.NoError(t, err)
	assert.True(t, proto.Equal(&publishingv1.Post{
		PostId: "p-1", Status: "scheduled", ScheduledFor: timestamppb.New(time.Date(2026, 10, 24, 15, 0, 0, 0, time.UTC)),
		Metadata: metadata,
	}, listed[0]), "got %v", listed[0])
	assert.True(t, proto.Equal(&publishingv1.Post{
		PostId: "p-4", Status: "partial", ScheduledFor: timestamppb.New(time.Date(2026, 10, 24, 10, 0, 0, 0, time.UTC)),
		Failure: "tiktok: TikTok creator has reached the daily posting limit.",
	}, listed[3]), "got %v", listed[3])
	assert.True(t, proto.Equal(&publishingv1.Post{PostId: "p-7", Title: "A DRAFT", Status: "draft"}, listed[6]), "got %v", listed[6])
	assert.Equal(t, url.Values{"fromDate": {"2026-10-23"}, "limit": {"500"}}, fake.queries["GET /v1/posts"])
}

func TestAListOfPostsWithNoPostsInItIsAnError(t *testing.T) {
	fake := newFake()
	fake.listed = answer{status: http.StatusOK, body: `{"pagination":{"page":1}}`}

	_, err := fake.service(t).Posts(t.Context(), at)

	require.ErrorContains(t, err, "zernio answered 200 to the list of posts")
}

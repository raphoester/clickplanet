package zernio_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/zernio"
)

const (
	oneAccountEach = `{"accounts":[{"_id":"acc-1","platform":"tiktok","isActive":true},{"_id":"acc-2","platform":"youtube","isActive":true}]}`
	onBoth         = `"platforms":[{"platform":"tiktok","status":"pending"},{"platform":"youtube","status":"pending"}]`
	kept           = `{"post":{"_id":"post-1","status":"scheduled","scheduledFor":"2026-10-12T16:40:00Z",` + onBoth + `},` +
		`"message":"Post scheduled successfully"}`
)

var at = time.Date(2026, 10, 12, 16, 40, 0, 0, time.UTC)

type zernioFake struct {
	accounts answer
	presign  answer
	upload   answer
	posts    answer
	listed   answer

	calls     []string
	queries   map[string]url.Values
	headers   map[string]http.Header
	uploaded  []byte
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
		upload:   answer{status: http.StatusOK},
		posts:    answer{status: http.StatusCreated, body: kept},
		listed:   answer{status: http.StatusOK, body: `{"posts":[]}`},
		headers:  map[string]http.Header{},
		queries:  map[string]url.Values{},
	}
}

func (f *zernioFake) serve(t *testing.T) *httptest.Server {
	t.Helper()

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			if f.presign.status != 0 {
				f.presign.write(w)
				return
			}
			answer{status: http.StatusOK, body: fmt.Sprintf(
				`{"uploadUrl":%q,"publicUrl":"https://media.example/clip-1.mp4","key":"temp/clip-1.mp4","expiresIn":3600}`,
				server.URL+"/storage/clip-1.mp4?signature=the-signature",
			)}.write(w)
		case "PUT /storage/clip-1.mp4":
			f.uploaded = raw
			f.upload.write(w)
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
	return server
}

func post(t *testing.T, platforms ...posts.Platform) posts.Post {
	t.Helper()

	return postOf(t, posts.StoryKey{Flag: "fr", Continent: "Europe"}, platforms...)
}

func postOf(t *testing.T, key posts.StoryKey, platforms ...posts.Platform) posts.Post {
	t.Helper()

	return postWith(t, key, nil, "", platforms...)
}

func postWith(t *testing.T, key posts.StoryKey, record posts.ClipRecord, teaser string, platforms ...posts.Platform) posts.Post {
	t.Helper()

	if len(platforms) == 0 {
		platforms = []posts.Platform{posts.TikTok, posts.YouTube}
	}
	video := filepath.Join(t.TempDir(), "clip-1.mp4")
	require.NoError(t, os.WriteFile(video, []byte("the video"), 0o600))
	clip, err := posts.ClipSnapshot{
		Pick: 1, Title: "FRANCE IS ATTACKING EUROPE", Teasers: []string{"FRANCE TAKES OVER EUROPE"}, Countries: []posts.TaggedCountry{{Code: "fr", Tag: "france"}},
		ContinentTag: "europe", Key: key, Video: posts.VideoFile(video), Record: record, TeaserNames: namesOf(teaser),
	}.Restore()
	require.NoError(t, err)

	planned := posts.Plan{Review: 4 * time.Hour, Platforms: platforms}.Posts([]posts.Clip{clip}, at.Add(-4*time.Hour), posts.LoadedDice(0))
	require.Equal(t, at, planned[0].TakeSnapshot().At)
	return planned[0]
}

// The one teaser's name, or none.
func namesOf(teaser string) []string {
	if teaser == "" {
		return nil
	}
	return []string{teaser}
}

// What a post of a clip with no record and no teaser names keeps.
var unnamedPost = map[string]any{
	"teaser": "unnamed", "callToAction": "join-the-battle", "scheduledAt": "2026-10-12T16:40:00Z",
	"runPosition": float64(1), "runSize": float64(1),
}

func schedule(t *testing.T, fake *zernioFake, platforms ...posts.Platform) (posts.Scheduled, error) {
	t.Helper()

	server := fake.serve(t)
	service, err := zernio.New(zernio.Config{APIKey: "the-key", BaseURL: server.URL}, server.Client())
	require.NoError(t, err)
	return service.Schedule(t.Context(), post(t, platforms...)) //nolint:wrapcheck // the test reads the service's own error.
}

func tiktokTarget(t *testing.T) map[string]any {
	t.Helper()

	return map[string]any{
		"platform": "tiktok", "accountId": "acc-1", "customContent": post(t).TakeSnapshot().TikTok.Caption,
		"platformSpecificData": map[string]any{
			"privacyLevel": "PUBLIC_TO_EVERYONE", "allowComment": true, "allowDuet": true, "allowStitch": true,
			"contentPreviewConfirmed": true, "expressConsentGiven": true,
		},
	}
}

func TestAClipIsUploadedOnceThenScheduledAsOnePostOnTikTokAndYouTube(t *testing.T) {
	fake := newFake()

	scheduled, err := schedule(t, fake)

	require.NoError(t, err)
	assert.Equal(t, posts.PostID("post-1"), scheduled.ID)
	assert.Empty(t, scheduled.Refused)
	assert.Equal(t, []string{"GET /v1/accounts", "POST /v1/media/presign", "PUT /storage/clip-1.mp4", "POST /v1/posts"}, fake.calls)
	assert.Equal(t, map[string]any{"filename": "clip-1.mp4", "contentType": "video/mp4", "size": float64(9)}, fake.presigned)
	assert.Equal(t, []byte("the video"), fake.uploaded)
	assert.Equal(t, "video/mp4", fake.headers["PUT /storage/clip-1.mp4"].Get("Content-Type"))
	youtube := post(t).TakeSnapshot().YouTube
	tags := make([]any, 0, len(youtube.Tags))
	for _, tag := range youtube.Tags {
		tags = append(tags, tag)
	}
	assert.Equal(t, map[string]any{
		"title":      "FRANCE IS ATTACKING EUROPE",
		"mediaItems": []any{map[string]any{"type": "video", "url": "https://media.example/clip-1.mp4"}},
		"platforms": []any{
			tiktokTarget(t),
			map[string]any{
				"platform": "youtube", "accountId": "acc-2", "customContent": youtube.Description,
				"platformSpecificData": map[string]any{
					"title": "FRANCE TAKES OVER EUROPE | CLICKPLANET", "visibility": "public", "madeForKids": false, "categoryId": "20",
				},
			},
		},
		"tags":         tags,
		"scheduledFor": "2026-10-12T16:40:00Z",
		"metadata": map[string]any{
			"storyFlag": "fr", "storyContinent": "Europe", "storyCountries": []any{"fr"}, "callToAction": "join-the-battle",
			"record": map[string]any{"schema": "clips.v1.PostRecord", "post": unnamedPost},
		},
	}, fake.sent)
}

func TestWhatAStoryTellsOfItsFlagAndItsDayAreKeptWithThePost(t *testing.T) {
	fake := newFake()
	server := fake.serve(t)
	service, err := zernio.New(zernio.Config{APIKey: "the-key", BaseURL: server.URL}, server.Client())
	require.NoError(t, err)

	_, err = service.Schedule(t.Context(), postOf(t, posts.StoryKey{Flag: "fr", Continent: "the world", Angle: posts.DayWon, Round: "1.5"}))

	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"storyFlag": "fr", "storyContinent": "the world", "storyAngle": "round", "storyRound": "1.5",
		"storyCountries": []any{"fr"}, "callToAction": "join-the-battle",
		"record": map[string]any{"schema": "clips.v1.PostRecord", "post": unnamedPost},
	}, fake.sent["metadata"])
}

func TestWhatThePostWasIsOneDocumentUnderOneKeyWithTheClipAsTheGeneratorWroteIt(t *testing.T) {
	fake := newFake()
	server := fake.serve(t)
	service, err := zernio.New(zernio.Config{APIKey: "the-key", BaseURL: server.URL}, server.Client())
	require.NoError(t, err)
	record := posts.ClipRecord(`{"format":"globe-replay","generator":"8dc31d2","common":{"durationSeconds":17.4,"music":"none",` +
		`"language":"en"},"globe":{"camera":"tour","look":"none","tilesChanged":5200,"tourStops":3},"variants":{"end-card":"podium"}}`)

	_, err = service.Schedule(t.Context(), postWith(t, posts.StoryKey{Flag: "pt", Continent: "the world"}, record, "takes-over-the-world"))

	require.NoError(t, err)
	metadata, ok := fake.sent["metadata"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, map[string]any{
		"schema": "clips.v1.PostRecord",
		"clip": map[string]any{
			"format": "globe-replay", "generator": "8dc31d2",
			"common":   map[string]any{"durationSeconds": 17.4, "music": "none", "language": "en"},
			"globe":    map[string]any{"camera": "tour", "look": "none", "tilesChanged": float64(5200), "tourStops": float64(3)},
			"variants": map[string]any{"end-card": "podium"},
		},
		"post": map[string]any{
			"teaser": "takes-over-the-world", "callToAction": "join-the-battle", "scheduledAt": "2026-10-12T16:40:00Z",
			"runPosition": float64(1), "runSize": float64(1),
		},
	}, metadata["record"])
	assert.Equal(t, "pt", metadata["storyFlag"])
}

func TestARecordThatIsNotAClipRecordUploadsNothing(t *testing.T) {
	fake := newFake()
	server := fake.serve(t)
	service, err := zernio.New(zernio.Config{APIKey: "the-key", BaseURL: server.URL}, server.Client())
	require.NoError(t, err)

	_, err = service.Schedule(t.Context(), postWith(t, posts.StoryKey{Flag: "fr", Continent: "Europe"}, posts.ClipRecord(`{"format":3}`), "invades"))

	require.Error(t, err)
	assert.Equal(t, []string{"GET /v1/accounts"}, fake.calls)
}

func TestAPostForTikTokAloneHasNoYouTubeTargetAndNoTags(t *testing.T) {
	fake := newFake()
	fake.posts = answer{status: http.StatusCreated, body: `{"post":{"_id":"post-1","status":"scheduled","platforms":[{"platform":"tiktok"}]}}`}

	scheduled, err := schedule(t, fake, posts.TikTok)

	require.NoError(t, err)
	assert.Equal(t, posts.Scheduled{ID: "post-1", Refused: map[posts.Platform]error{}}, scheduled)
	assert.Equal(t, []any{tiktokTarget(t)}, fake.sent["platforms"])
	assert.NotContains(t, fake.sent, "tags")
}

func TestTheKeyGoesToZernioAndNeverToTheStorage(t *testing.T) {
	fake := newFake()

	_, err := schedule(t, fake)

	require.NoError(t, err)
	for _, call := range []string{"GET /v1/accounts", "POST /v1/media/presign", "POST /v1/posts"} {
		assert.Equal(t, "Bearer the-key", fake.headers[call].Get("Authorization"), call)
	}
	assert.Empty(t, fake.headers["PUT /storage/clip-1.mp4"].Get("Authorization"))
}

func TestTheSameVideoIsSentUnderTheSameKeySoARunMadeAgainPostsItOnce(t *testing.T) {
	first, second := newFake(), newFake()
	second.posts = answer{status: http.StatusOK, body: kept}

	_, err := schedule(t, first)
	require.NoError(t, err)
	scheduled, err := schedule(t, second)

	require.NoError(t, err)
	assert.Equal(t, posts.PostID("post-1"), scheduled.ID)
	key := first.headers["POST /v1/posts"].Get("Idempotency-Key")
	assert.Len(t, key, 2*sha256.Size)
	assert.Equal(t, key, second.headers["POST /v1/posts"].Get("Idempotency-Key"))
}

func TestAnAnswerWithFieldsAddedLaterIsStillRead(t *testing.T) {
	fake := newFake()
	fake.accounts = answer{status: http.StatusOK, body: `{"accounts":[` +
		`{"_id":"acc-1","platform":"tiktok","profileId":"p-1","isActive":true,"addedLater":1},` +
		`{"_id":"acc-2","platform":"youtube","profileId":"p-1","isActive":true,"addedLater":1}` +
		`],"hasAnalyticsAccess":false,"addedLater":1}`}
	fake.posts = answer{status: http.StatusCreated, body: `{"post":{"_id":"post-1","status":"scheduled","addedLater":1,"platforms":[` +
		`{"platform":"tiktok","status":"pending","addedLater":1,` +
		`"accountId":{"_id":"acc-1","platform":"tiktok","username":"@clickplanet","isActive":true,"addedLater":1}},` +
		`{"platform":"youtube","status":"pending","addedLater":1,` +
		`"accountId":{"_id":"acc-2","platform":"youtube","username":"@clickplanet","isActive":true,"addedLater":1}}` +
		`]},"addedLater":1}`}

	scheduled, err := schedule(t, fake)

	require.NoError(t, err)
	assert.Equal(t, posts.PostID("post-1"), scheduled.ID)
	assert.Empty(t, scheduled.Refused)
}

func TestTheSamePostAnsweredAgainWithFieldsAddedLaterIsStillRead(t *testing.T) {
	fake := newFake()
	fake.posts = answer{status: http.StatusOK, body: `{"post":{"_id":"post-1","status":"scheduled","addedLater":1,` + onBoth + `},` +
		`"message":"Post already exists (idempotent retry)","addedLater":1}`}

	scheduled, err := schedule(t, fake)

	require.NoError(t, err)
	assert.Equal(t, posts.PostID("post-1"), scheduled.ID)
	assert.Empty(t, scheduled.Refused)
}

func TestAPlatformThePostZernioKeptIsNotOnIsRefused(t *testing.T) {
	fake := newFake()
	fake.posts = answer{status: http.StatusOK, body: `{"post":{"_id":"post-1","status":"scheduled","platforms":[{"platform":"tiktok"}]},` +
		`"message":"Post already exists (idempotent retry)"}`}

	scheduled, err := schedule(t, fake)

	require.NoError(t, err)
	assert.Equal(t, posts.PostID("post-1"), scheduled.ID)
	require.Len(t, scheduled.Refused, 1)
	require.ErrorContains(t, scheduled.Refused[posts.YouTube], "zernio kept the post without it")
}

func TestAPostZernioDidNotKeepAsScheduledIsAFailure(t *testing.T) {
	for _, status := range []string{"published", "publishing", "draft", "failed", ""} {
		t.Run(status, func(t *testing.T) {
			fake := newFake()
			fake.posts = answer{status: http.StatusCreated, body: fmt.Sprintf(`{"post":{"_id":"post-1","status":%q}}`, status)}

			_, err := schedule(t, fake)

			require.ErrorContains(t, err, "not \"scheduled\"")
		})
	}
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

			_, err := schedule(t, fake)

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

			_, err := schedule(t, fake)

			var busy *posts.BusyError
			require.Error(t, err)
			assert.NotErrorAs(t, err, &busy)
			assert.ErrorContains(t, err, fmt.Sprintf("zernio answered %d to the post: ", refused.status))
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

			scheduled, err := schedule(t, fake)

			require.NoError(t, err)
			assert.Equal(t, posts.PostID("post-1"), scheduled.ID)
			require.Len(t, scheduled.Refused, 1)
			require.ErrorContains(t, scheduled.Refused[posts.YouTube], "connected youtube accounts, and the job posts to exactly one")
			assert.Equal(t, []any{tiktokTarget(t)}, fake.sent["platforms"])
			assert.NotContains(t, fake.sent, "tags")
		})
	}
}

func TestNoAccountOnAnyPlatformIsNoPostAndNoUpload(t *testing.T) {
	fake := newFake()
	fake.accounts = answer{status: http.StatusOK, body: `{"accounts":[]}`}

	scheduled, err := schedule(t, fake)

	require.NoError(t, err)
	assert.Empty(t, scheduled.ID)
	assert.Len(t, scheduled.Refused, 2)
	assert.Equal(t, []string{"GET /v1/accounts"}, fake.calls)
}

func TestAListOfAccountsZernioRefusesIsAnError(t *testing.T) {
	fake := newFake()
	fake.accounts = answer{status: http.StatusUnauthorized, body: `{"error":"Invalid API key"}`}

	_, err := schedule(t, fake)

	require.ErrorContains(t, err, "zernio answered 401 to the list of accounts: Invalid API key")
	assert.Equal(t, []string{"GET /v1/accounts"}, fake.calls)
}

func TestAFailedUploadSchedulesNothing(t *testing.T) {
	fake := newFake()
	fake.upload = answer{status: http.StatusForbidden}

	_, err := schedule(t, fake)

	require.ErrorContains(t, err, "zernio answered 403 to the upload of the video")
	assert.NotContains(t, fake.calls, "POST /v1/posts")
}

func TestAnUploadThatCannotBeReachedIsBusyAndNeverSaysItsAddress(t *testing.T) {
	fake := newFake()
	fake.presign = answer{status: http.StatusOK, body: `{"uploadUrl":"http://127.0.0.1:1/storage?signature=the-signature","publicUrl":"https://media.example/clip-1.mp4"}`}

	_, err := schedule(t, fake)

	var busy *posts.BusyError
	require.ErrorAs(t, err, &busy)
	assert.NotContains(t, err.Error(), "the-signature")
	assert.NotContains(t, err.Error(), "127.0.0.1:1/storage")
}

func TestAnErrorNeverSaysTheKey(t *testing.T) {
	fake := newFake()
	fake.posts = answer{status: http.StatusUnauthorized, body: `{"error":"Invalid API key"}`}

	_, err := schedule(t, fake)

	require.Error(t, err)
	assert.NotContains(t, err.Error(), "the-key")
}

func TestAnEmptyKeyIsRefusedWhenTheServiceIsBuilt(t *testing.T) {
	_, err := zernio.New(zernio.Config{}, http.DefaultClient)

	require.Error(t, err)
}

func TestAVideoThatIsNotThereIsAnError(t *testing.T) {
	fake := newFake()
	server := fake.serve(t)
	service, err := zernio.New(zernio.Config{APIKey: "the-key", BaseURL: server.URL}, server.Client())
	require.NoError(t, err)
	clip, err := posts.ClipSnapshot{
		Pick: 1, Title: "T", Teasers: []string{"T2"}, Key: posts.StoryKey{Flag: "fr", Continent: "Europe"}, Video: posts.VideoFile(filepath.Join(t.TempDir(), "gone.mp4")),
	}.Restore()
	require.NoError(t, err)

	_, err = service.Schedule(t.Context(), posts.Plan{Review: 4 * time.Hour, Platforms: []posts.Platform{posts.TikTok}}.Posts([]posts.Clip{clip}, at, posts.LoadedDice(0))[0])

	require.ErrorContains(t, err, "failed to read the video")
	assert.Empty(t, fake.calls)
}

func told(t *testing.T, fake *zernioFake, since time.Time) ([]posts.Telling, error) {
	t.Helper()

	server := fake.serve(t)
	service, err := zernio.New(zernio.Config{APIKey: "the-key", BaseURL: server.URL}, server.Client())
	require.NoError(t, err)
	return service.Told(t.Context(), since) //nolint:wrapcheck // the test reads the service's own error.
}

func TestTheStoriesToldAreThoseOfThePostsWithAKeyOnEveryPlatform(t *testing.T) {
	fake := newFake()
	fake.listed = answer{status: http.StatusOK, body: `{"posts":[` +
		`{"_id":"p-1","title":"ALGERIA IS INVADING LIBYA","status":"scheduled","scheduledFor":"2026-10-14T05:15:00Z",` +
		`"metadata":{"storyFlag":"dz","storyContinent":"Africa","storyCountries":["ly","dz"],"usageCounted":true}},` +
		`{"_id":"p-2","title":"PALESTINE IS KICKING ROMANIA OUT OF KAZAKHSTAN","status":"published","scheduledFor":"2026-10-13T21:09:00.000Z",` +
		`"metadata":{"storyFlag":"ps","storyContinent":"Central Asia","storyCountries":["kz","ps",3,"ro"]},"addedLater":1,` +
		`"platforms":[{"platform":"instagram","status":"published"}]},` +
		`{"_id":"p-3","status":"published","scheduledFor":"2026-10-13T20:00:00Z","content":"a post made by hand"},` +
		`{"_id":"p-4","status":"cancelled","scheduledFor":"2026-10-13T23:09:00Z","metadata":{"storyFlag":"pt","storyContinent":"Europe","storyCountries":"pt"}},` +
		`{"_id":"p-5","title":"FRANCE IS KICKING MOROCCO OUT OF LIBYA","status":"failed","scheduledFor":"2026-10-14T07:15:00Z",` +
		`"metadata":{"storyFlag":"fr","storyContinent":"Africa"},"platforms":[` +
		`{"platform":"tiktok","status":"failed","errorMessage":"TikTok creator has reached the daily posting limit."},` +
		`{"platform":"instagram","status":"published"}]}` +
		`],"pagination":{"page":1}}`}

	tellings, err := told(t, fake, time.Date(2026, 10, 13, 0, 57, 0, 0, time.UTC))

	require.NoError(t, err)
	assert.Equal(t, []posts.Telling{
		{
			Key: posts.StoryKey{Flag: "dz", Continent: "Africa"}, Countries: []string{"ly", "dz"},
			At: time.Date(2026, 10, 14, 5, 15, 0, 0, time.UTC), Title: "ALGERIA IS INVADING LIBYA", State: "scheduled",
		},
		{
			Key: posts.StoryKey{Flag: "ps", Continent: "Central Asia"}, Countries: []string{"kz", "ps", "ro"},
			At: time.Date(2026, 10, 13, 21, 9, 0, 0, time.UTC), Title: "PALESTINE IS KICKING ROMANIA OUT OF KAZAKHSTAN", State: "published",
		},
		{Key: posts.StoryKey{Flag: "pt", Continent: "Europe"}, At: time.Date(2026, 10, 13, 23, 9, 0, 0, time.UTC), State: "cancelled"},
		{
			Key: posts.StoryKey{Flag: "fr", Continent: "Africa"}, At: time.Date(2026, 10, 14, 7, 15, 0, 0, time.UTC),
			Title: "FRANCE IS KICKING MOROCCO OUT OF LIBYA", State: "failed",
			Failure: "tiktok: TikTok creator has reached the daily posting limit.",
		},
	}, tellings)
	assert.Equal(t, url.Values{"fromDate": {"2026-10-13"}, "limit": {"500"}}, fake.queries["GET /v1/posts"])
	assert.Equal(t, "Bearer the-key", fake.headers["GET /v1/posts"].Get("Authorization"))
}

func TestAPostOfADayWonOrAHoldingIsReadWithItsAngle(t *testing.T) {
	fake := newFake()
	fake.listed = answer{status: http.StatusOK, body: `{"posts":[` +
		`{"_id":"p-1","title":"FRANCE WINS DAY 5","status":"scheduled","scheduledFor":"2026-10-14T05:15:00Z",` +
		`"metadata":{"storyFlag":"fr","storyContinent":"the world","storyAngle":"round","storyRound":"1.5"}},` +
		`{"_id":"p-2","title":"CROATIA HOLDS 80% OF EUROPE","status":"scheduled","scheduledFor":"2026-10-14T07:15:00Z",` +
		`"metadata":{"storyFlag":"hr","storyContinent":"Europe","storyAngle":"holding"}}` +
		`],"pagination":{"page":1}}`}

	tellings, err := told(t, fake, time.Date(2026, 10, 13, 0, 57, 0, 0, time.UTC))

	require.NoError(t, err)
	require.Len(t, tellings, 2)
	assert.Equal(t, posts.StoryKey{Flag: "fr", Continent: "the world", Angle: posts.DayWon, Round: "1.5"}, tellings[0].Key)
	assert.Equal(t, posts.StoryKey{Flag: "hr", Continent: "Europe", Angle: posts.Holding}, tellings[1].Key)
}

func TestAListOfPostsZernioRefusesIsAnError(t *testing.T) {
	fake := newFake()
	fake.listed = answer{status: http.StatusUnauthorized, body: `{"error":"Invalid API key"}`}

	_, err := told(t, fake, at)

	require.ErrorContains(t, err, "zernio answered 401 to the list of posts: Invalid API key")
}

func TestAListOfPostsWithNoPostsInItIsAnError(t *testing.T) {
	fake := newFake()
	fake.listed = answer{status: http.StatusOK, body: `{}`}

	_, err := told(t, fake, at)

	require.Error(t, err)
}

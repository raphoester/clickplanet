package zernio_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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
	oneAccount = `{"accounts":[{"_id":"acc-1","platform":"tiktok","isActive":true}]}`
	kept       = `{"post":{"_id":"post-1","status":"scheduled","scheduledFor":"2026-10-12T16:40:00Z"},"message":"Post scheduled successfully"}`
	caption    = "FRANCE IS ATTACKING EUROPE. Who stops them? 👇\nclickplanet.lol\n#clickplanet #europe"
)

var at = time.Date(2026, 10, 12, 16, 40, 0, 0, time.UTC)

type zernioFake struct {
	accounts answer
	presign  answer
	upload   answer
	posts    answer

	calls     []string
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
		accounts: answer{status: http.StatusOK, body: oneAccount},
		upload:   answer{status: http.StatusOK},
		posts:    answer{status: http.StatusCreated, body: kept},
		headers:  map[string]http.Header{},
	}
}

func (f *zernioFake) serve(t *testing.T) *httptest.Server {
	t.Helper()

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := r.Method + " " + r.URL.Path
		f.calls = append(f.calls, call)
		f.headers[call] = r.Header.Clone()
		raw, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		switch call {
		case "GET /v1/accounts":
			assert.Equal(t, "tiktok", r.URL.Query().Get("platform"))
			assert.Equal(t, "true", r.URL.Query().Get("excludeHidden"))
			f.accounts.write(w)
		case "POST /v1/media/presign":
			assert.NoError(t, json.Unmarshal(raw, &f.presigned))
			if f.presign.status != 0 {
				f.presign.write(w)
				return
			}
			answer{status: http.StatusOK, body: fmt.Sprintf(
				`{"uploadUrl":%q,"publicUrl":"https://media.example/clip-1.mp4","key":"temp/clip-1.mp4","expiresIn":3600}`,
				server.URL+"/storage/clip-1.mp4?signature=the-signature")}.write(w)
		case "PUT /storage/clip-1.mp4":
			f.uploaded = raw
			f.upload.write(w)
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

func post(t *testing.T) posts.Post {
	t.Helper()

	video := filepath.Join(t.TempDir(), "clip-1.mp4")
	require.NoError(t, os.WriteFile(video, []byte("the video"), 0o600))
	clip, err := posts.ClipSnapshot{Pick: 1, Title: "FRANCE IS ATTACKING EUROPE", Caption: caption, Video: posts.VideoFile(video)}.Restore()
	require.NoError(t, err)

	planned := posts.Cadence{Review: 4 * time.Hour}.Posts([]posts.Clip{clip}, at.Add(-4*time.Hour))
	require.Equal(t, at, planned[0].TakeSnapshot().At)
	return planned[0]
}

func schedule(t *testing.T, fake *zernioFake) (posts.PostID, error) {
	t.Helper()

	server := fake.serve(t)
	service, err := zernio.New(zernio.Config{APIKey: "the-key", BaseURL: server.URL}, server.Client())
	require.NoError(t, err)
	return service.Schedule(t.Context(), post(t)) //nolint:wrapcheck // the test reads the service's own error.
}

func TestAClipIsUploadedThenScheduledOnTheOneTikTokAccount(t *testing.T) {
	fake := newFake()

	id, err := schedule(t, fake)

	require.NoError(t, err)
	assert.Equal(t, posts.PostID("post-1"), id)
	assert.Equal(t, []string{"GET /v1/accounts", "POST /v1/media/presign", "PUT /storage/clip-1.mp4", "POST /v1/posts"}, fake.calls)
	assert.Equal(t, map[string]any{"filename": "clip-1.mp4", "contentType": "video/mp4", "size": float64(9)}, fake.presigned)
	assert.Equal(t, []byte("the video"), fake.uploaded)
	assert.Equal(t, "video/mp4", fake.headers["PUT /storage/clip-1.mp4"].Get("Content-Type"))
	assert.Equal(t, map[string]any{
		"title":        "FRANCE IS ATTACKING EUROPE",
		"content":      caption,
		"mediaItems":   []any{map[string]any{"type": "video", "url": "https://media.example/clip-1.mp4"}},
		"platforms":    []any{map[string]any{"platform": "tiktok", "accountId": "acc-1"}},
		"scheduledFor": "2026-10-12T16:40:00Z",
		"tiktokSettings": map[string]any{
			"privacyLevel": "PUBLIC_TO_EVERYONE", "allowComment": true, "allowDuet": true, "allowStitch": true,
			"contentPreviewConfirmed": true, "expressConsentGiven": true,
		},
	}, fake.sent)
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
	id, err := schedule(t, second)

	require.NoError(t, err)
	assert.Equal(t, posts.PostID("post-1"), id)
	key := first.headers["POST /v1/posts"].Get("Idempotency-Key")
	assert.Len(t, key, 2*sha256.Size)
	assert.Equal(t, key, second.headers["POST /v1/posts"].Get("Idempotency-Key"))
}

func TestAnAnswerWithFieldsAddedLaterIsStillRead(t *testing.T) {
	fake := newFake()
	fake.accounts = answer{status: http.StatusOK, body: `{"accounts":[{"_id":"acc-1","platform":"tiktok","profileId":"p-1","isActive":true,"addedLater":1}],"hasAnalyticsAccess":false,"addedLater":1}`}
	fake.posts = answer{status: http.StatusCreated, body: `{"post":{"_id":"post-1","status":"scheduled","addedLater":1,` +
		`"platforms":[{"platform":"tiktok","status":"pending","addedLater":1,` +
		`"accountId":{"_id":"acc-1","platform":"tiktok","username":"@clickplanet","isActive":true,"addedLater":1}}]},"addedLater":1}`}

	id, err := schedule(t, fake)

	require.NoError(t, err)
	assert.Equal(t, posts.PostID("post-1"), id)
}

func TestTheSamePostAnsweredAgainWithFieldsAddedLaterIsStillRead(t *testing.T) {
	fake := newFake()
	fake.posts = answer{status: http.StatusOK, body: `{"post":{"_id":"post-1","status":"scheduled","addedLater":1},"message":"Post already exists (idempotent retry)","addedLater":1}`}

	id, err := schedule(t, fake)

	require.NoError(t, err)
	assert.Equal(t, posts.PostID("post-1"), id)
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

func TestNoTikTokAccountOrMoreThanOneIsRefusedBeforeAnyUpload(t *testing.T) {
	for name, accounts := range map[string]string{
		"none":         `{"accounts":[]}`,
		"disconnected": `{"accounts":[{"_id":"acc-1","platform":"tiktok","isActive":false}]}`,
		"two":          `{"accounts":[{"_id":"acc-1","platform":"tiktok","isActive":true},{"_id":"acc-2","platform":"tiktok","isActive":true}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			fake := newFake()
			fake.accounts = answer{status: http.StatusOK, body: accounts}

			_, err := schedule(t, fake)

			require.ErrorContains(t, err, "exactly one")
			assert.Equal(t, []string{"GET /v1/accounts"}, fake.calls)
		})
	}
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
	clip, err := posts.ClipSnapshot{Pick: 1, Title: "T", Caption: "C", Video: posts.VideoFile(filepath.Join(t.TempDir(), "gone.mp4"))}.Restore()
	require.NoError(t, err)

	_, err = service.Schedule(t.Context(), posts.Cadence{Review: 4 * time.Hour}.Posts([]posts.Clip{clip}, at)[0])

	require.ErrorContains(t, err, "failed to read the video")
	assert.Empty(t, fake.calls)
}

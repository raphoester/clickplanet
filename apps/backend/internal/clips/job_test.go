package clips_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips"
)

type sentPost struct {
	Title        string `json:"title"`
	Content      string `json:"content"`
	ScheduledFor string `json:"scheduledFor"`
	MediaItems   []struct {
		URL string `json:"url"`
	} `json:"mediaItems"`
}

// A zernio that takes every post, and keeps what it was sent.
func zernio(t *testing.T, posts *[]sentPost) *httptest.Server {
	t.Helper()

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		switch r.Method + " " + r.URL.Path {
		case "GET /accounts":
			_, _ = fmt.Fprint(w, `{"accounts":[{"_id":"acc-1","platform":"tiktok","isActive":true}]}`)
		case "POST /media/presign":
			var asked struct {
				Filename string `json:"filename"`
			}
			assert.NoError(t, json.Unmarshal(raw, &asked))
			_, _ = fmt.Fprintf(w, `{"uploadUrl":%q,"publicUrl":%q}`, server.URL+"/storage/"+asked.Filename, "https://media.example/"+asked.Filename)
		case "PUT /storage/clip-1.mp4", "PUT /storage/clip-2.mp4":
		case "POST /posts":
			var post sentPost
			assert.NoError(t, json.Unmarshal(raw, &post))
			*posts = append(*posts, post)
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprintf(w, `{"post":{"_id":"post-%d","status":"scheduled"}}`, len(*posts))
		default:
			t.Errorf("zernio was asked %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func rendered(t *testing.T, folder string, pick int, title string) {
	t.Helper()

	clip := fmt.Sprintf(`{"pick": %d, "title": %q, "caption": %q, "tags": ["#clickplanet"]}`, pick, title, title+" 👇\n#clickplanet")
	require.NoError(t, os.WriteFile(filepath.Join(folder, fmt.Sprintf("clip-%d.json", pick)), []byte(clip), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(folder, fmt.Sprintf("clip-%d.mp4", pick)), []byte(title), 0o600))
}

func TestARunSchedulesEveryClipOfTheFolderHoursAheadAndSaysSoInTheSummary(t *testing.T) {
	var posts []sentPost
	server := zernio(t, &posts)
	folder := t.TempDir()
	rendered(t, folder, 2, "FRANCE IS ATTACKING EUROPE")
	rendered(t, folder, 1, "KOSOVO IS ATTACKING THE MIDDLE EAST")
	var config clips.Config
	config.Zernio.APIKey = "the-key"
	config.Zernio.BaseURL = server.URL
	var summary, logs bytes.Buffer
	before := time.Now()

	err := clips.Schedule(t.Context(), folder, config, &summary, slog.New(slog.NewTextHandler(&logs, nil)))

	require.NoError(t, err)
	require.Len(t, posts, 2)
	assert.Equal(t, "KOSOVO IS ATTACKING THE MIDDLE EAST", posts[0].Title)
	assert.Equal(t, "KOSOVO IS ATTACKING THE MIDDLE EAST 👇\n#clickplanet", posts[0].Content)
	assert.Equal(t, "https://media.example/clip-1.mp4", posts[0].MediaItems[0].URL)
	assert.Equal(t, "FRANCE IS ATTACKING EUROPE", posts[1].Title)

	first, err := time.Parse(time.RFC3339, posts[0].ScheduledFor)
	require.NoError(t, err)
	second, err := time.Parse(time.RFC3339, posts[1].ScheduledFor)
	require.NoError(t, err)
	assert.WithinDuration(t, before.Add(4*time.Hour), first, 2*time.Minute)
	assert.False(t, first.Before(before.Add(4*time.Hour)))
	assert.Equal(t, 2*time.Hour, second.Sub(first))

	assert.Contains(t, summary.String(), "| KOSOVO IS ATTACKING THE MIDDLE EAST | "+first.UTC().Format("Mon 2 Jan, 15:04 UTC")+" | scheduled |")
	assert.Contains(t, summary.String(), "| FRANCE IS ATTACKING EUROPE | "+second.UTC().Format("Mon 2 Jan, 15:04 UTC")+" | scheduled |")
	assert.Contains(t, logs.String(), `msg="scheduled a clip" clip="KOSOVO IS ATTACKING THE MIDDLE EAST"`)
	assert.NotContains(t, logs.String()+summary.String(), "the-key")
}

func TestARunThatZernioRefusesFailsAndTheSummarySaysWhy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, `{"error":"Invalid API key"}`)
	}))
	t.Cleanup(server.Close)
	folder := t.TempDir()
	rendered(t, folder, 1, "KOSOVO IS ATTACKING THE MIDDLE EAST")
	var config clips.Config
	config.Zernio.APIKey = "the-key"
	config.Zernio.BaseURL = server.URL
	var summary, logs bytes.Buffer

	err := clips.Schedule(t.Context(), folder, config, &summary, slog.New(slog.NewTextHandler(&logs, nil)))

	require.ErrorContains(t, err, "1 of 1")
	assert.Contains(t, summary.String(), "**not scheduled**: zernio answered 401 to GET /accounts")
	assert.Contains(t, logs.String(), `level=ERROR msg="failed to schedule a clip"`)
}

func TestChoosingWritesThePicksOfTheBestStories(t *testing.T) {
	shortlist := filepath.Join(t.TempDir(), "shortlist.json")
	require.NoError(t, os.WriteFile(shortlist, []byte(`{"stories":[{"title":"BEST"},{"title":"SECOND"},{"title":"THIRD"}]}`), 0o600))
	var picks, logs bytes.Buffer

	err := clips.Choose(t.Context(), shortlist, clips.Config{}, &picks, slog.New(slog.NewTextHandler(&logs, nil)))

	require.NoError(t, err)
	assert.JSONEq(t, "[1,2]", picks.String())
	assert.Contains(t, logs.String(), `msg="chose a story" pick=2 title=SECOND`)
}

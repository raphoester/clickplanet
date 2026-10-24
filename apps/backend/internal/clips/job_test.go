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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips"
)

type sentPost struct {
	Title        string         `json:"title"`
	ScheduledFor string         `json:"scheduledFor"`
	Metadata     map[string]any `json:"metadata"`
	MediaItems   []struct {
		URL string `json:"url"`
	} `json:"mediaItems"`
	Platforms []sentTarget `json:"platforms"`
}

type sentTarget struct {
	Platform      string `json:"platform"`
	CustomContent string `json:"customContent"`
}

type keptPost struct {
	ID           string         `json:"_id"`
	Status       string         `json:"status"`
	ScheduledFor string         `json:"scheduledFor"`
	Metadata     map[string]any `json:"metadata"`
	Platforms    []keptTarget   `json:"platforms"`
}

type keptTarget struct {
	Platform string `json:"platform"`
}

func zernio(t *testing.T, posts *[]sentPost) *httptest.Server {
	t.Helper()

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")

		switch r.Method + " " + r.URL.Path {
		case "GET /v1/accounts":
			_, _ = fmt.Fprint(w, `{"accounts":[{"_id":"acc-1","platform":"tiktok","isActive":true},{"_id":"acc-2","platform":"youtube","isActive":true}]}`)
		case "POST /v1/media/presign":
			var asked struct {
				Filename string `json:"filename"`
			}
			assert.NoError(t, json.Unmarshal(raw, &asked))
			_, _ = fmt.Fprintf(w, `{"uploadUrl":%q,"publicUrl":%q}`, server.URL+"/storage/"+asked.Filename, "https://media.example/"+asked.Filename)
		case "PUT /storage/clip-1.mp4", "PUT /storage/clip-2.mp4":
		case "POST /v1/posts":
			var post sentPost
			assert.NoError(t, json.Unmarshal(raw, &post))
			*posts = append(*posts, post)
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprintf(w, `{"post":{"_id":"post-%d","status":"scheduled","platforms":[{"platform":"tiktok"},{"platform":"youtube"}]}}`, len(*posts))
		case "GET /v1/posts":
			kept := make([]keptPost, 0, len(*posts))
			for i, post := range *posts {
				targets := make([]keptTarget, 0, len(post.Platforms))
				for _, target := range post.Platforms {
					targets = append(targets, keptTarget{Platform: target.Platform})
				}
				kept = append(kept, keptPost{
					ID: fmt.Sprintf("post-%d", i+1), Status: "scheduled", ScheduledFor: post.ScheduledFor, Metadata: post.Metadata, Platforms: targets,
				})
			}
			assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{"posts": kept}))
		default:
			t.Errorf("zernio was asked %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func rendered(t *testing.T, folder string, pick int, title, flag string, countries ...string) {
	t.Helper()

	tagged := make([]string, 0, len(countries))
	for _, country := range countries {
		tagged = append(tagged, fmt.Sprintf(`{"code": %q, "tag": %q}`, country, country))
	}
	clip := fmt.Sprintf(`{"pick": %d, "title": %q, "teasers": [%q], "key": {"flag": %q, "continent": "Africa"}, "countries": [%s], "continentTag": "africa"}`,
		pick, title, "SAID ANOTHER WAY: "+title, flag, strings.Join(tagged, ", "))
	require.NoError(t, os.WriteFile(filepath.Join(folder, fmt.Sprintf("clip-%d.json", pick)), []byte(clip), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(folder, fmt.Sprintf("clip-%d.mp4", pick)), []byte(title), 0o600))
}

func TestARunSchedulesEveryClipOfTheFolderAtOnceAnHourApartAndReportsEachOne(t *testing.T) {
	var posts []sentPost
	server := zernio(t, &posts)
	folder := t.TempDir()
	rendered(t, folder, 2, "FRANCE IS ATTACKING EUROPE", "fr", "fr", "de")
	rendered(t, folder, 1, "KOSOVO IS ATTACKING THE MIDDLE EAST", "xk")
	var config clips.Config
	config.Zernio.APIKey = "the-key"
	config.Zernio.BaseURL = server.URL
	var logs bytes.Buffer
	before := time.Now()

	report, err := clips.Schedule(t.Context(), folder, config, slog.New(slog.NewTextHandler(&logs, nil)))

	require.NoError(t, err)
	require.Len(t, posts, 2)
	assert.Equal(t, "KOSOVO IS ATTACKING THE MIDDLE EAST", posts[0].Title)
	assert.Equal(t, "xk", posts[0].Metadata["storyFlag"])
	assert.Equal(t, "Africa", posts[0].Metadata["storyContinent"])
	assert.NotEmpty(t, posts[0].Metadata["callToAction"])
	require.Len(t, posts[0].Platforms, 2)
	tiktok, youtube := posts[0].Platforms[0], posts[0].Platforms[1]
	assert.Equal(t, "tiktok", tiktok.Platform)
	assert.True(t, strings.HasPrefix(tiktok.CustomContent, "SAID ANOTHER WAY: KOSOVO IS ATTACKING THE MIDDLE EAST | CLICKPLANET\n"))
	assert.Contains(t, tiktok.CustomContent, "👉 https://clickplanet.lol\n#clickplanet ")
	assert.True(t, strings.HasSuffix(tiktok.CustomContent, " #africa"))
	assert.Equal(t, "youtube", youtube.Platform)
	assert.True(t, strings.HasSuffix(youtube.CustomContent, " #africa"))
	assert.Equal(t, "https://media.example/clip-1.mp4", posts[0].MediaItems[0].URL)
	assert.Equal(t, "FRANCE IS ATTACKING EUROPE", posts[1].Title)
	assert.Equal(t, []any{"fr", "de"}, posts[1].Metadata["storyCountries"])

	first, err := time.Parse(time.RFC3339, posts[0].ScheduledFor)
	require.NoError(t, err)
	second, err := time.Parse(time.RFC3339, posts[1].ScheduledFor)
	require.NoError(t, err)
	assert.WithinDuration(t, before.Add(5*time.Minute), first, 2*time.Minute)
	assert.False(t, first.Before(before.Add(5*time.Minute)))
	assert.Equal(t, time.Hour, second.Sub(first))

	require.Len(t, report, 2)
	assert.Equal(t, "KOSOVO IS ATTACKING THE MIDDLE EAST", report[0].Title)
	assert.Equal(t, clips.Report{{
		Title: "FRANCE IS ATTACKING EUROPE", At: report[1].At, Post: "post-2",
		Platforms: []clips.PlatformOutcome{{Platform: "tiktok"}, {Platform: "youtube"}},
	}}, report[1:])
	assert.True(t, first.Equal(report[0].At))
	assert.True(t, second.Equal(report[1].At))
	assert.True(t, report.Complete())
	assert.Contains(t, logs.String(), `msg="scheduled a clip" clip="KOSOVO IS ATTACKING THE MIDDLE EAST" platform=youtube`)
	assert.NotContains(t, logs.String(), "the-key")
}

func TestARunThatZernioRefusesReportsWhyForEachClip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method+" "+r.URL.Path == "GET /v1/posts" {
			_, _ = fmt.Fprint(w, `{"posts":[]}`)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, `{"error":"Invalid API key"}`)
	}))
	t.Cleanup(server.Close)
	folder := t.TempDir()
	rendered(t, folder, 1, "KOSOVO IS ATTACKING THE MIDDLE EAST", "xk")
	var config clips.Config
	config.Zernio.APIKey = "the-key"
	config.Zernio.BaseURL = server.URL
	var logs bytes.Buffer

	report, err := clips.Schedule(t.Context(), folder, config, slog.New(slog.NewTextHandler(&logs, nil)))

	require.NoError(t, err)
	require.Len(t, report, 1)
	assert.False(t, report.Complete())
	require.Len(t, report[0].Platforms, 2)
	assert.Contains(t, report[0].Platforms[1].Failure, "zernio answered 401 to the list of accounts: Invalid API key")
	assert.Contains(t, logs.String(), `level=ERROR msg="failed to schedule a clip"`)
}

func TestARunThatCannotReadWhatThePlatformsHoldSendsNothing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method+" "+r.URL.Path != "GET /v1/posts" {
			t.Errorf("zernio was asked %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, `{"error":"Invalid API key"}`)
	}))
	t.Cleanup(server.Close)
	folder := t.TempDir()
	rendered(t, folder, 1, "KOSOVO IS ATTACKING THE MIDDLE EAST", "xk")

	_, err := clips.Schedule(t.Context(), folder, configFor(server), slog.New(slog.DiscardHandler))

	require.ErrorContains(t, err, "zernio answered 401 to the list of posts: Invalid API key")
}

func shortlist(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "shortlist.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"stories":[`+
		`{"title":"ALGERIA STRIKES BACK","key":{"flag":"dz","continent":"Africa"}},`+
		`{"title":"ALGERIA VS FRANCE","key":{"flag":"dz","continent":"Africa"}},`+
		`{"title":"PALESTINE IS KICKING ROMANIA OUT OF KAZAKHSTAN","key":{"flag":"ps","continent":"Central Asia"}},`+
		`{"title":"FRANCE IS ATTACKING EUROPE","key":{"flag":"fr","continent":"Europe"}}]}`), 0o600))
	return path
}

func configFor(server *httptest.Server) clips.Config {
	var config clips.Config
	config.Zernio.APIKey = "the-key"
	config.Zernio.BaseURL = server.URL
	return config
}

// The picks, each in an arm of the length experiment: the coins are the system's, so which arm is not known.
func picksOf(t *testing.T, assignments []clips.Assignment) []clips.Pick {
	t.Helper()

	picks := make([]clips.Pick, 0, len(assignments))
	for _, assignment := range assignments {
		assert.Contains(t, []string{"as-chosen", "short"}, assignment.Variants["length"])
		picks = append(picks, assignment.Pick)
	}
	return picks
}

func TestChoosingGivesTheBestStoriesOfDifferentFlagsOrContinents(t *testing.T) {
	var posts []sentPost
	var logs bytes.Buffer

	picks, err := clips.Choose(t.Context(), shortlist(t), configFor(zernio(t, &posts)), slog.New(slog.NewTextHandler(&logs, nil)))

	require.NoError(t, err)
	assert.Equal(t, []clips.Pick{1, 3}, picksOf(t, picks))
	assert.Contains(t, logs.String(), `msg="skipped a story told already" pick=2 title="ALGERIA VS FRANCE"`)
}

func TestARunAfterADayOfPostsOnEveryPlatformChoosesNothing(t *testing.T) {
	posts := day(14, "tiktok", "youtube")
	var logs bytes.Buffer

	picks, err := clips.Choose(t.Context(), shortlist(t), configFor(zernio(t, &posts)), slog.New(slog.NewTextHandler(&logs, nil)))

	require.NoError(t, err)
	assert.Empty(t, picks)
	assert.Contains(t, logs.String(), `msg="every platform has posted its day already"`)
}

func TestTikTokTakesFourteenADayLeavingOneOfItsFifteenForAPostByHand(t *testing.T) {
	for given, want := range map[int]int{10: 2, 13: 1, 14: 0} {
		t.Run(fmt.Sprint(given), func(t *testing.T) {
			posts := day(given, "tiktok")
			config := configFor(zernio(t, &posts))
			config.Platforms = []clips.Platform{"tiktok"}

			picks, err := clips.Choose(t.Context(), shortlist(t), config, slog.New(slog.DiscardHandler))

			require.NoError(t, err)
			assert.Len(t, picks, want)
		})
	}
}

// count posts on the platforms, an hour apart, the newest an hour ago.
func day(count int, platforms ...string) []sentPost {
	posts := make([]sentPost, count)
	for i := range posts {
		posts[i].ScheduledFor = time.Now().Add(-time.Duration(i+1) * time.Hour).UTC().Format(time.RFC3339)
		for _, platform := range platforms {
			posts[i].Platforms = append(posts[i].Platforms, sentTarget{Platform: platform})
		}
	}
	return posts
}

func TestAStoryOneRunScheduledIsNotChosenByTheNext(t *testing.T) {
	var posts []sentPost
	server := zernio(t, &posts)
	folder := t.TempDir()
	rendered(t, folder, 1, "ALGERIA STRIKES BACK", "dz")
	_, err := clips.Schedule(t.Context(), folder, configFor(server), slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	picks, err := clips.Choose(t.Context(), shortlist(t), configFor(server), slog.New(slog.DiscardHandler))

	require.NoError(t, err)
	assert.Equal(t, []clips.Pick{3, 4}, picksOf(t, picks))
}

func TestAFlagTheLastRunShowedTwiceWaitsThoughNeitherClipWasAboutIt(t *testing.T) {
	var posts []sentPost
	server := zernio(t, &posts)
	folder := t.TempDir()
	rendered(t, folder, 1, "ALGERIA VS PALAU", "dz", "dz", "pw")
	rendered(t, folder, 2, "FRANCE VS PALAU", "fr", "fr", "pw")
	_, err := clips.Schedule(t.Context(), folder, configFor(server), slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "shortlist.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"stories":[`+
		`{"title":"PALAU IS ATTACKING EUROPE","key":{"flag":"pw","continent":"Europe"},"countries":["pw","pl"]},`+
		`{"title":"PALESTINE IS KICKING ROMANIA OUT OF KAZAKHSTAN","key":{"flag":"ps","continent":"Central Asia"}}]}`), 0o600))

	picks, err := clips.Choose(t.Context(), path, configFor(server), slog.New(slog.DiscardHandler))

	require.NoError(t, err)
	assert.Equal(t, []clips.Pick{2}, picksOf(t, picks))
}

func renderedIn(t *testing.T, folder string, pick int, title, variants string) {
	t.Helper()

	clip := fmt.Sprintf(`{"pick": %d, "title": %q, "teasers": [%q], "key": {"flag": "dz", "continent": "Africa"}, `+
		`"record": {"format": "globe-replay", "variants": %s}}`, pick, title, "SAID ANOTHER WAY: "+title, variants)
	require.NoError(t, os.WriteFile(filepath.Join(folder, fmt.Sprintf("clip-%d.json", pick)), []byte(clip), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(folder, fmt.Sprintf("clip-%d.mp4", pick)), []byte(title), 0o600))
}

func TestTheArmsAClipWasMadeInAreKeptWithItsPostAndReported(t *testing.T) {
	var posts []sentPost
	folder := t.TempDir()
	renderedIn(t, folder, 1, "ALGERIA STRIKES BACK", `{"length": "short"}`)

	report, err := clips.Schedule(t.Context(), folder, configFor(zernio(t, &posts)), slog.New(slog.DiscardHandler))

	require.NoError(t, err)
	require.Len(t, posts, 1)
	record, ok := posts[0].Metadata["record"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, map[string]any{"length": "short"}, record["clip"].(map[string]any)["variants"])
	assert.Equal(t, clips.Variants{"length": "short"}, report[0].Variants)
}

func TestAClipInAnArmThatDoesNotRunIsNotPosted(t *testing.T) {
	var posts []sentPost
	folder := t.TempDir()
	renderedIn(t, folder, 1, "ALGERIA STRIKES BACK", `{"length": "long"}`)

	_, err := clips.Schedule(t.Context(), folder, configFor(zernio(t, &posts)), slog.New(slog.DiscardHandler))

	require.ErrorContains(t, err, `the experiment "length" has no arm "long"`)
	assert.Empty(t, posts)
}

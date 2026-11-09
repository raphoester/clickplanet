package clips_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1/publishingv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// The backend as the job reaches it: its publishing module, a fake of it on httptest, behind an ssh on the PATH that
// sends each call there with curl, which prints nothing of a call it refuses, as busybox's wget does.
type backend struct {
	t       *testing.T
	mu      sync.Mutex
	storage *httptest.Server
	// What each platform takes now: 3 on TikTok and 12 on YouTube when nil.
	left map[publishingv1.Platform]uint32
	// Why every post is refused, or none.
	failure string
	// The calendar's now: a slot is the first whole minute 5 minutes from it, an hour after the last post.
	now     time.Time
	posted  []*publishingv1.SchedulePostRequest
	uploads []string
}

func serve(t *testing.T, now time.Time) *backend {
	t.Helper()

	b := &backend{t: t, now: now}
	b.storage = httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		b.mu.Lock()
		b.uploads = append(b.uploads, r.URL.Path)
		b.mu.Unlock()
	}))
	t.Cleanup(b.storage.Close)

	_, handler := publishingv1connect.NewAdminServiceHandler(b)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	folder := t.TempDir()
	script := fmt.Sprintf("#!/bin/sh\nmethod=${3##*/}\nexec curl -sf -X POST -H 'Content-Type: application/json' --data-binary @- %q/publishing.v1.AdminService/\"$method\"\n", server.URL)
	require.NoError(t, os.WriteFile(filepath.Join(folder, "ssh"), []byte(script), 0o700)) //nolint:gosec // a script the test runs.
	t.Setenv("PATH", folder+string(os.PathListSeparator)+os.Getenv("PATH"))
	return b
}

func (b *backend) config() clips.Config {
	var config clips.Config
	config.Replay.SSH = "deploy@vps.example"
	return config
}

func (b *backend) CreateMediaUpload(
	_ context.Context, req *connect.Request[publishingv1.CreateMediaUploadRequest],
) (*connect.Response[publishingv1.CreateMediaUploadResponse], error) {
	return connect.NewResponse(&publishingv1.CreateMediaUploadResponse{
		UploadUrl: b.storage.URL + "/temp/" + req.Msg.GetFileName(), MediaUrl: "https://media.example/" + req.Msg.GetFileName(),
	}), nil
}

func (b *backend) GetRoom(
	_ context.Context, req *connect.Request[publishingv1.GetRoomRequest],
) (*connect.Response[publishingv1.GetRoomResponse], error) {
	left := b.left
	if left == nil {
		left = map[publishingv1.Platform]uint32{publishingv1.Platform_PLATFORM_TIKTOK: 3, publishingv1.Platform_PLATFORM_YOUTUBE: 12}
	}
	answer := &publishingv1.GetRoomResponse{}
	for _, platform := range req.Msg.GetPlatforms() {
		answer.Rooms = append(answer.Rooms, &publishingv1.PlatformRoom{Platform: platform, Left: left[platform]})
	}
	return connect.NewResponse(answer), nil
}

func (b *backend) PlanPost(
	_ context.Context, req *connect.Request[publishingv1.PlanPostRequest],
) (*connect.Response[publishingv1.PlanPostResponse], error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	at := b.now.Add(5 * time.Minute).Truncate(time.Minute).Add(time.Minute)
	if len(b.posted) > 0 {
		at = b.posted[len(b.posted)-1].GetSlot().GetAt().AsTime().Add(time.Hour)
	}
	return connect.NewResponse(&publishingv1.PlanPostResponse{Slot: &publishingv1.Slot{
		At: timestamppb.New(at), Platforms: req.Msg.GetPlatforms(),
	}}), nil
}

func (b *backend) SchedulePost(
	_ context.Context, req *connect.Request[publishingv1.SchedulePostRequest],
) (*connect.Response[publishingv1.SchedulePostResponse], error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.failure != "" {
		return connect.NewResponse(&publishingv1.SchedulePostResponse{Outcome: &publishingv1.SchedulePostResponse_Failed{
			Failed: &publishingv1.Failed{Reason: b.failure},
		}}), nil
	}
	b.posted = append(b.posted, req.Msg)
	return connect.NewResponse(&publishingv1.SchedulePostResponse{Outcome: &publishingv1.SchedulePostResponse_Scheduled{
		Scheduled: &publishingv1.Scheduled{PostId: fmt.Sprintf("post-%d", len(b.posted))},
	}}), nil
}

func (b *backend) ListPosts(
	context.Context, *connect.Request[publishingv1.ListPostsRequest],
) (*connect.Response[publishingv1.ListPostsResponse], error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	answer := &publishingv1.ListPostsResponse{}
	for i, post := range b.posted {
		answer.Posts = append(answer.Posts, &publishingv1.Post{
			PostId: fmt.Sprintf("post-%d", i+1), Title: post.GetTitle(), ScheduledFor: post.GetSlot().GetAt(),
			Status: "scheduled", Metadata: post.GetMetadata(),
		})
	}
	return connect.NewResponse(answer), nil
}

// Today at 15:00 UTC.
func afternoon() *cptime.FixedClock {
	return cptime.NewFixedClock(time.Now().UTC().Truncate(24 * time.Hour).Add(15 * time.Hour))
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

func TestARunPostsEveryClipOfTheFolderThroughTheBackendBestFirstAndReportsEachOne(t *testing.T) {
	clock := afternoon()
	b := serve(t, clock.Now())
	folder := t.TempDir()
	rendered(t, folder, 2, "FRANCE IS ATTACKING EUROPE", "fr", "fr", "de")
	rendered(t, folder, 1, "KOSOVO IS ATTACKING THE MIDDLE EAST", "xk")
	var logs bytes.Buffer

	report, err := clips.Schedule(t.Context(), folder, b.config(), clock, slog.New(slog.NewTextHandler(&logs, nil)))

	require.NoError(t, err)
	require.Len(t, b.posted, 2)
	best, second := b.posted[0], b.posted[1]
	assert.Equal(t, "KOSOVO IS ATTACKING THE MIDDLE EAST", best.GetTitle())
	kept := best.GetMetadata().AsMap()
	assert.Equal(t, "xk", kept["storyFlag"])
	assert.Equal(t, "Africa", kept["storyContinent"])
	assert.NotEmpty(t, kept["callToAction"])
	assert.Equal(t, []publishingv1.Platform{publishingv1.Platform_PLATFORM_TIKTOK, publishingv1.Platform_PLATFORM_YOUTUBE},
		best.GetSlot().GetPlatforms())
	assert.True(t, strings.HasPrefix(best.GetTiktok().GetCaption(), "SAID ANOTHER WAY: KOSOVO IS ATTACKING THE MIDDLE EAST | CLICKPLANET\n"))
	assert.Contains(t, best.GetTiktok().GetCaption(), "👉 https://clickplanet.lol\n#clickplanet ")
	assert.True(t, strings.HasSuffix(best.GetTiktok().GetCaption(), " #africa"))
	assert.True(t, strings.HasSuffix(best.GetYoutube().GetDescription(), " #africa"))
	assert.Equal(t, "https://media.example/clip-1.mp4", best.GetMediaUrl())
	assert.Len(t, best.GetIdempotencyKey(), 64, "the SHA-256 of the video")
	assert.Equal(t, []string{"/temp/clip-1.mp4", "/temp/clip-2.mp4"}, b.uploads, "each video straight to the storage")
	assert.Equal(t, "FRANCE IS ATTACKING EUROPE", second.GetTitle())
	assert.Equal(t, []any{"fr", "de"}, second.GetMetadata().AsMap()["storyCountries"])
	// TikTok takes a run's best clip alone.
	assert.Equal(t, []publishingv1.Platform{publishingv1.Platform_PLATFORM_YOUTUBE}, second.GetSlot().GetPlatforms())

	first := best.GetSlot().GetAt().AsTime()
	assert.Equal(t, time.Hour, second.GetSlot().GetAt().AsTime().Sub(first), "the next one asked once the first was made")
	record, _ := kept["record"].(map[string]any)
	facts, _ := record["post"].(map[string]any)
	assert.Equal(t, first.Format(time.RFC3339), facts["scheduledAt"], "the record keeps the slot it was planned at")

	require.Len(t, report, 2)
	assert.Equal(t, clips.Report{
		{
			Title: "KOSOVO IS ATTACKING THE MIDDLE EAST", At: first, Post: "post-1",
			Platforms: []clips.PlatformOutcome{{Platform: "tiktok"}, {Platform: "youtube"}},
		},
		{
			Title: "FRANCE IS ATTACKING EUROPE", At: first.Add(time.Hour), Post: "post-2",
			Platforms: []clips.PlatformOutcome{{Platform: "youtube"}},
		},
	}, report)
	assert.True(t, report.Complete())
	assert.Contains(t, logs.String(), `msg="scheduled a clip" clip="KOSOVO IS ATTACKING THE MIDDLE EAST" platform=youtube`)
}

func TestARunTheBackendRefusesReportsWhyForEachClip(t *testing.T) {
	clock := afternoon()
	b := serve(t, clock.Now())
	b.failure = "zernio answered 401 to the list of accounts: Invalid API key"
	folder := t.TempDir()
	rendered(t, folder, 1, "KOSOVO IS ATTACKING THE MIDDLE EAST", "xk")
	var logs bytes.Buffer

	report, err := clips.Schedule(t.Context(), folder, b.config(), clock, slog.New(slog.NewTextHandler(&logs, nil)))

	require.NoError(t, err)
	require.Len(t, report, 1)
	assert.False(t, report.Complete())
	require.Len(t, report[0].Platforms, 2)
	assert.Equal(t, "zernio answered 401 to the list of accounts: Invalid API key", report[0].Platforms[1].Failure)
	assert.Contains(t, logs.String(), `level=ERROR msg="failed to schedule a clip"`)
}

// A day won and a country called up after it, which the run tells first, and the front of the war it would ask the
// editor about.
func findings(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "findings.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"fronts":[{"countries":[{"code":"tr","continent":"Middle East","takers":{"fr":5000}}],`+
		`"flags":[{"code":"fr","continent":"Europe","took":5000}]}],"seasonal":[`+
		`{"title":"ALGERIA WINS DAY 17","key":{"flag":"dz","continent":"the world","angle":"round","round":"0.17"}},`+
		`{"title":"POLSKA: 14 PUNKTÓW STRATY","key":{"flag":"pl","continent":"the world","angle":"callup","round":"0.17"}}]}`), 0o600))
	return path
}

// The stories told, in no experiment. With no ANTHROPIC_API_KEY the run is its seasonal stories alone.
func toldBy(t *testing.T, choice clips.Choice) []clips.Story {
	t.Helper()

	stories := make([]clips.Story, 0, len(choice.Assigned))
	for i, assignment := range choice.Assigned {
		assert.Empty(t, assignment.Variants)
		assert.Equal(t, clips.Pick(i+1), assignment.Pick)
		stories = append(stories, assignment.Story)
	}
	return stories
}

func TestChoosingWithNoEditorKeyTellsTheSeasonalStoriesAloneAndSaysWhy(t *testing.T) {
	var logs bytes.Buffer
	config := serve(t, time.Now()).config()
	config.PerRun = 3

	picks, err := clips.Choose(t.Context(), findings(t), config, slog.New(slog.NewTextHandler(&logs, nil)))

	require.NoError(t, err)
	assert.Equal(t, []clips.Story{{Seasonal: 1}, {Seasonal: 2}}, toldBy(t, picks))
	assert.Contains(t, logs.String(), `level=WARN msg="the editor gave no answer, and the run is its seasonal stories alone" why="no ANTHROPIC_API_KEY"`)
}

func TestARunWhenEveryPlatformHasPostedItsDayChoosesNothing(t *testing.T) {
	b := serve(t, time.Now())
	b.left = map[publishingv1.Platform]uint32{}
	var logs bytes.Buffer

	picks, err := clips.Choose(t.Context(), findings(t), b.config(), slog.New(slog.NewTextHandler(&logs, nil)))

	require.NoError(t, err)
	assert.Empty(t, picks.Assigned)
	assert.False(t, picks.Editorial.Asked)
	assert.Contains(t, logs.String(), `msg="every platform has posted its day already"`)
}

func TestTikTokTakesTheBestClipOfARunWhileItHasRoom(t *testing.T) {
	for left, want := range map[uint32]int{3: 1, 1: 1, 0: 0} {
		t.Run(fmt.Sprint(left), func(t *testing.T) {
			b := serve(t, time.Now())
			b.left = map[publishingv1.Platform]uint32{publishingv1.Platform_PLATFORM_TIKTOK: left}
			config := b.config()
			config.Platforms = []clips.Platform{"tiktok"}

			picks, err := clips.Choose(t.Context(), findings(t), config, slog.New(slog.DiscardHandler))

			require.NoError(t, err)
			assert.Len(t, picks.Assigned, want)
		})
	}
}

func TestARunThatCannotReachTheBackendChoosesNothingAndSaysWhy(t *testing.T) {
	folder := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(folder, "ssh"), []byte("#!/bin/sh\necho 'Permission denied (publickey)' >&2\nexit 255\n"), 0o700)) //nolint:gosec // a script the test runs.
	t.Setenv("PATH", folder+string(os.PathListSeparator)+os.Getenv("PATH"))
	var config clips.Config
	config.Replay.SSH = "deploy@vps.example"

	_, err := clips.Choose(t.Context(), findings(t), config, slog.New(slog.DiscardHandler))

	require.ErrorContains(t, err, "Permission denied (publickey)")
}

func TestADayOneRunScheduledIsNotToldByTheNext(t *testing.T) {
	clock := afternoon()
	b := serve(t, clock.Now())
	folder := t.TempDir()
	clip := `{"pick": 1, "title": "ALGERIA WINS DAY 17", "teasers": ["ALGERIA TAKES DAY 17"], "key": {"flag": "dz", "continent": "the world", "angle": "round", "round": "0.17"}}`
	require.NoError(t, os.WriteFile(filepath.Join(folder, "clip-1.json"), []byte(clip), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(folder, "clip-1.mp4"), []byte("the video"), 0o600))
	_, err := clips.Schedule(t.Context(), folder, b.config(), clock, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	picks, err := clips.Choose(t.Context(), findings(t), b.config(), slog.New(slog.DiscardHandler))

	require.NoError(t, err)
	assert.Equal(t, []clips.Story{{Seasonal: 2}}, toldBy(t, picks))
}

func renderedIn(t *testing.T, folder string, pick int, title, variants string) {
	t.Helper()

	clip := fmt.Sprintf(`{"pick": %d, "title": %q, "teasers": [%q], "key": {"flag": "dz", "continent": "Africa"}, `+
		`"record": {"format": "globe-replay", "common": {"durationSeconds": 10}, "variants": %s}}`,
		pick, title, "SAID ANOTHER WAY: "+title, variants)
	require.NoError(t, os.WriteFile(filepath.Join(folder, fmt.Sprintf("clip-%d.json", pick)), []byte(clip), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(folder, fmt.Sprintf("clip-%d.mp4", pick)), []byte(title), 0o600))
}

func TestAClipInAnExperimentThatDoesNotRunIsNotPosted(t *testing.T) {
	clock := afternoon()
	b := serve(t, clock.Now())
	folder := t.TempDir()
	renderedIn(t, folder, 1, "ALGERIA STRIKES BACK", `{"sound": "trending"}`)

	_, err := clips.Schedule(t.Context(), folder, b.config(), clock, slog.New(slog.DiscardHandler))

	require.ErrorContains(t, err, `no experiment "sound" runs`)
	assert.Empty(t, b.posted)
	assert.Empty(t, b.uploads)
}

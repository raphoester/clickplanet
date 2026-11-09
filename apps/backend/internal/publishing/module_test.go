package publishing_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1/publishingv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

var (
	tiktok  = publishingv1.Platform_PLATFORM_TIKTOK
	youtube = publishingv1.Platform_PLATFORM_YOUTUBE
)

// Zernio as the module sees it: one account a platform, the posts made, and the ones it is asked to make.
type zernioFake struct {
	mu     sync.Mutex
	posted []string
	sent   []map[string]any
}

func (f *zernioFake) serve(t *testing.T) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()

		assert.Equal(t, "Bearer the-key", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "GET /v1/accounts":
			_, _ = fmt.Fprint(w, `{"accounts":[{"_id":"acc-1","platform":"tiktok","isActive":true},{"_id":"acc-2","platform":"youtube","isActive":true}]}`)
		case "POST /v1/media/presign":
			_, _ = fmt.Fprint(w, `{"uploadUrl":"https://storage.example/clip-1.mp4?signature=s","publicUrl":"https://media.example/clip-1.mp4"}`)
		case "GET /v1/posts":
			_, _ = fmt.Fprintf(w, `{"posts":[%s]}`, joined(f.posted))
		case "POST /v1/posts":
			raw, _ := io.ReadAll(r.Body)
			var sent map[string]any
			assert.NoError(t, json.Unmarshal(raw, &sent))
			f.sent = append(f.sent, sent)
			post := fmt.Sprintf(`{"_id":"post-%d","title":"made","status":"scheduled","scheduledFor":%q,"metadata":%s,"platforms":[{"platform":"youtube","status":"pending"}]}`,
				len(f.sent), sent["scheduledFor"], mustJSON(t, sent["metadata"]))
			f.posted = append(f.posted, post)
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprintf(w, `{"post":%s}`, post)
		default:
			t.Errorf("zernio was asked %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func joined(posts []string) string {
	all := ""
	for i, post := range posts {
		if i > 0 {
			all += ","
		}
		all += post
	}
	return all
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()

	written, err := json.Marshal(value)
	require.NoError(t, err)
	return string(written)
}

func listen(t *testing.T) net.Listener {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	return listener
}

// The module alone in a server, and a client of its service on the console listener once it serves.
func run(t *testing.T, config publishing.Config) publishingv1connect.AdminServiceClient {
	t.Helper()

	public, console, internal := listen(t), listen(t), listen(t)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- cpbootstrap.RunOn(ctx, cpbootstrap.Options{
			Server: cpbootstrap.ServerConfig{
				BindAddress: public.Addr().String(), ConsoleBindAddress: console.Addr().String(),
				InternalBindAddress: internal.Addr().String(),
			},
			Logger:  slog.New(slog.DiscardHandler),
			Modules: []cpbootstrap.Module{publishing.NewModule(config)},
		}, public, console, internal)
	}()
	t.Cleanup(func() {
		cancel()
		require.NoError(t, <-done)
	})

	client := publishingv1connect.NewAdminServiceClient(http.DefaultClient, "http://"+console.Addr().String())
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		// Served is an answer, or the refusal of a module with no key; not a listener that is not there yet.
		_, err := client.ListPosts(t.Context(), connect.NewRequest(&publishingv1.ListPostsRequest{Since: timestamppb.Now()}))
		assert.True(c, err == nil || connect.CodeOf(err) == connect.CodeFailedPrecondition, "%v", err)
	}, time.Minute, 100*time.Millisecond, "the module serves once the server runs")
	return client
}

// Tomorrow at an hour of the day: in the hours of both platforms at 15:00, of YouTube alone at 12:00.
func tomorrowAt(hour int) time.Time {
	day := time.Now().UTC().AddDate(0, 0, 1)
	return time.Date(day.Year(), day.Month(), day.Day(), hour, 0, 0, 0, time.UTC)
}

func TestAJobPlansAPostSchedulesItAtItsSlotAndReadsItBackOnTheConsoleListener(t *testing.T) {
	fake := &zernioFake{}
	client := run(t, publishing.Config{Zernio: publishing.ZernioConfig{APIKey: "the-key", BaseURL: fake.serve(t)}})

	upload, err := client.CreateMediaUpload(t.Context(), connect.NewRequest(&publishingv1.CreateMediaUploadRequest{
		FileName: "clip-1.mp4", ContentType: "video/mp4", SizeBytes: 80 << 20,
	}))
	require.NoError(t, err)
	assert.Equal(t, "https://media.example/clip-1.mp4", upload.Msg.GetMediaUrl())

	room, err := client.GetRoom(t.Context(), connect.NewRequest(&publishingv1.GetRoomRequest{Platforms: []publishingv1.Platform{tiktok, youtube}}))
	require.NoError(t, err)
	assert.Equal(t, []uint32{3, 12}, []uint32{room.Msg.GetRooms()[0].GetLeft(), room.Msg.GetRooms()[1].GetLeft()})

	plan, err := client.PlanPost(t.Context(), connect.NewRequest(&publishingv1.PlanPostRequest{
		Platforms: []publishingv1.Platform{tiktok, youtube}, NotBefore: timestamppb.New(tomorrowAt(15)),
	}))
	require.NoError(t, err)
	assert.Equal(t, tomorrowAt(15), plan.Msg.GetSlot().GetAt().AsTime())
	assert.Equal(t, []publishingv1.Platform{tiktok, youtube}, plan.Msg.GetSlot().GetPlatforms())

	// At 10:00 TikTok is out of its hours: left out, or waited for with by.
	early := &publishingv1.PlanPostRequest{Platforms: []publishingv1.Platform{tiktok, youtube}, NotBefore: timestamppb.New(tomorrowAt(10))}
	plan, err = client.PlanPost(t.Context(), connect.NewRequest(early))
	require.NoError(t, err)
	assert.Equal(t, []publishingv1.Platform{youtube}, plan.Msg.GetSlot().GetPlatforms())
	early.By = timestamppb.New(tomorrowAt(20))
	plan, err = client.PlanPost(t.Context(), connect.NewRequest(early))
	require.NoError(t, err)
	assert.Equal(t, tomorrowAt(14), plan.Msg.GetSlot().GetAt().AsTime())
	assert.Equal(t, []publishingv1.Platform{tiktok, youtube}, plan.Msg.GetSlot().GetPlatforms())

	metadata, err := structpb.NewStruct(map[string]any{"storyFlag": "fr"})
	require.NoError(t, err)
	at := &publishingv1.Slot{At: timestamppb.New(tomorrowAt(12)), Platforms: []publishingv1.Platform{youtube}}
	request := &publishingv1.SchedulePostRequest{
		Slot: at, MediaUrl: upload.Msg.GetMediaUrl(), IdempotencyKey: "9b2a", Title: "FRANCE VS TURKEY",
		Youtube: &publishingv1.YouTubeWords{Title: "FRANCE VS TURKEY | CLICKPLANET", Tags: []string{"map"}}, Metadata: metadata,
	}
	scheduled, err := client.SchedulePost(t.Context(), connect.NewRequest(request))
	require.NoError(t, err)
	assert.Equal(t, "post-1", scheduled.Msg.GetScheduled().GetPostId())
	assert.Empty(t, scheduled.Msg.GetScheduled().GetRefusals())

	again, err := client.SchedulePost(t.Context(), connect.NewRequest(request))
	require.NoError(t, err)
	assert.Contains(t, again.Msg.GetTaken().GetReason(), "no longer open", "the post now takes the slot")

	listed, err := client.ListPosts(t.Context(), connect.NewRequest(&publishingv1.ListPostsRequest{Since: timestamppb.Now()}))
	require.NoError(t, err)
	require.Len(t, listed.Msg.GetPosts(), 1)
	assert.Equal(t, "fr", listed.Msg.GetPosts()[0].GetMetadata().GetFields()["storyFlag"].GetStringValue())
	assert.Equal(t, tomorrowAt(12), listed.Msg.GetPosts()[0].GetScheduledFor().AsTime())
}

func TestATikTokCarouselGoesToZernioAsPhotosWithTheVideoForYouTube(t *testing.T) {
	fake := &zernioFake{}
	client := run(t, publishing.Config{Zernio: publishing.ZernioConfig{APIKey: "the-key", BaseURL: fake.serve(t)}})

	_, err := client.SchedulePost(t.Context(), connect.NewRequest(&publishingv1.SchedulePostRequest{
		Slot:     &publishingv1.Slot{At: timestamppb.New(tomorrowAt(15)), Platforms: []publishingv1.Platform{tiktok, youtube}},
		MediaUrl: "https://media.example/teaser.mp4", IdempotencyKey: "f00d", Title: "RING WAR",
		Tiktok:          &publishingv1.TikTokWords{Caption: "Close a ring around any land.", Title: "RING WAR · SATURDAY 20:00 UTC"},
		Youtube:         &publishingv1.YouTubeWords{Title: "RING WAR | CLICKPLANET"},
		TiktokPhotoUrls: []string{"https://media.example/card-1.jpg", "https://media.example/card-2.jpg"},
	}))

	require.NoError(t, err)
	require.Len(t, fake.sent, 1)
	targets, ok := fake.sent[0]["platforms"].([]any)
	require.True(t, ok)
	carousel, ok := targets[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "RING WAR · SATURDAY 20:00 UTC", carousel["customContent"])
	assert.Len(t, carousel["customMedia"], 2)

	_, err = client.SchedulePost(t.Context(), connect.NewRequest(&publishingv1.SchedulePostRequest{
		Slot:     &publishingv1.Slot{At: timestamppb.New(tomorrowAt(17)), Platforms: []publishingv1.Platform{tiktok}},
		MediaUrl: "https://media.example/teaser.mp4", IdempotencyKey: "f00d", Tiktok: &publishingv1.TikTokWords{Caption: "Close a ring."},
		TiktokPhotoUrls: []string{"https://media.example/card-1.jpg"},
	}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.ErrorContains(t, err, "the photos have no title")
}

func TestAWrongRequestIsInvalidArgument(t *testing.T) {
	client := run(t, publishing.Config{Zernio: publishing.ZernioConfig{APIKey: "the-key", BaseURL: (&zernioFake{}).serve(t)}})

	_, err := client.SchedulePost(t.Context(), connect.NewRequest(&publishingv1.SchedulePostRequest{}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = client.SchedulePost(t.Context(), connect.NewRequest(&publishingv1.SchedulePostRequest{
		Slot: &publishingv1.Slot{Platforms: []publishingv1.Platform{tiktok}}, MediaUrl: "https://media.example/clip-1.mp4", IdempotencyKey: "9b2a",
	}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.ErrorContains(t, err, "no time")
	assert.ErrorContains(t, err, "tiktok with no caption")

	_, err = client.GetRoom(t.Context(), connect.NewRequest(&publishingv1.GetRoomRequest{
		Platforms: []publishingv1.Platform{publishingv1.Platform_PLATFORM_UNSPECIFIED},
	}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = client.CreateMediaUpload(t.Context(), connect.NewRequest(&publishingv1.CreateMediaUploadRequest{
		FileName: "clip.png", ContentType: "image/png", SizeBytes: 1,
	}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = client.ListPosts(t.Context(), connect.NewRequest(&publishingv1.ListPostsRequest{}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestWithNoKeyEveryCallIsFailedPreconditionAndTheServerStillRuns(t *testing.T) {
	client := run(t, publishing.Config{})

	_, err := client.GetRoom(t.Context(), connect.NewRequest(&publishingv1.GetRoomRequest{Platforms: []publishingv1.Platform{tiktok}}))

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	assert.ErrorContains(t, err, "no key of the posting service")
}

func TestABaseURLThatIsNoHTTPAddressIsRefused(t *testing.T) {
	require.NoError(t, publishing.Config{}.Validate())
	require.NoError(t, publishing.Config{Zernio: publishing.ZernioConfig{BaseURL: "http://127.0.0.1:8090/api"}}.Validate())

	assert.ErrorContains(t, publishing.Config{Zernio: publishing.ZernioConfig{BaseURL: "zernio.com"}}.Validate(), "publishing.zernio.baseURL")
}

package promo_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1/promov1connect"
	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1/publishingv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo"
)

var (
	tiktok  = publishingv1.Platform_PLATFORM_TIKTOK
	youtube = publishingv1.Platform_PLATFORM_YOUTUBE
)

// The backend as the job reaches it: promo's ListTeasers and the publishing module, fakes of them on httptest, behind an
// ssh on the PATH that sends each call there with curl, which prints nothing of a call it refuses, as busybox's wget does.
type backend struct {
	promov1connect.UnimplementedAdminServiceHandler
	publishing

	listed []*promov1.Teaser
}

type publishing struct {
	publishingv1connect.UnimplementedAdminServiceHandler

	mu      sync.Mutex
	storage *httptest.Server
	// The battles the posts already tease.
	teased []string
	// The platforms that take a post by its battle, in the calendar's eyes.
	open    []publishingv1.Platform
	planned []*publishingv1.PlanPostRequest
	posted  []*publishingv1.SchedulePostRequest
	uploads []string
}

func serve(t *testing.T, listed ...*promov1.Teaser) *backend {
	t.Helper()

	b := &backend{listed: listed}
	b.open = []publishingv1.Platform{tiktok, youtube}
	b.storage = httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		b.mu.Lock()
		b.uploads = append(b.uploads, r.URL.Path)
		b.mu.Unlock()
	}))
	t.Cleanup(b.storage.Close)

	mux := http.NewServeMux()
	mux.Handle(promov1connect.NewAdminServiceHandler(b))
	mux.Handle(publishingv1connect.NewAdminServiceHandler(&b.publishing))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	folder := t.TempDir()
	script := fmt.Sprintf("#!/bin/sh\nmethod=${3##*8081/}\nexec curl -sf -X POST -H 'Content-Type: application/json' --data-binary @- %q/\"$method\"\n", server.URL)
	require.NoError(t, os.WriteFile(filepath.Join(folder, "ssh"), []byte(script), 0o700)) //nolint:gosec // a script the test runs.
	t.Setenv("PATH", folder+string(os.PathListSeparator)+os.Getenv("PATH"))
	return b
}

var config = promo.JobConfig{SSH: "deploy@vps.example"}

func (b *backend) ListTeasers(
	_ context.Context, req *connect.Request[promov1.ListTeasersRequest],
) (*connect.Response[promov1.ListTeasersResponse], error) {
	if req.Msg.GetWithin().AsDuration() != 36*time.Hour {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	return connect.NewResponse(&promov1.ListTeasersResponse{Teasers: b.listed}), nil
}

func (p *publishing) ListPosts(
	context.Context, *connect.Request[publishingv1.ListPostsRequest],
) (*connect.Response[publishingv1.ListPostsResponse], error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	answer := &publishingv1.ListPostsResponse{}
	for _, battle := range p.teased {
		metadata, _ := structpb.NewStruct(map[string]any{"teaserBattle": battle})
		answer.Posts = append(answer.Posts, &publishingv1.Post{PostId: "post-" + battle, Metadata: metadata})
	}
	return connect.NewResponse(answer), nil
}

func (p *publishing) CreateMediaUpload(
	_ context.Context, req *connect.Request[publishingv1.CreateMediaUploadRequest],
) (*connect.Response[publishingv1.CreateMediaUploadResponse], error) {
	return connect.NewResponse(&publishingv1.CreateMediaUploadResponse{
		UploadUrl: p.storage.URL + "/temp/" + req.Msg.GetFileName(), MediaUrl: "https://media.example/" + req.Msg.GetFileName(),
	}), nil
}

// From not_before, on the platforms asked when every one of them is open.
func (p *publishing) PlanPost(
	_ context.Context, req *connect.Request[publishingv1.PlanPostRequest],
) (*connect.Response[publishingv1.PlanPostResponse], error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.planned = append(p.planned, req.Msg)
	for _, platform := range req.Msg.GetPlatforms() {
		if !slices.Contains(p.open, platform) {
			return connect.NewResponse(&publishingv1.PlanPostResponse{}), nil
		}
	}
	return connect.NewResponse(&publishingv1.PlanPostResponse{Slot: &publishingv1.Slot{
		At: req.Msg.GetNotBefore(), Platforms: req.Msg.GetPlatforms(),
	}}), nil
}

func (p *publishing) SchedulePost(
	_ context.Context, req *connect.Request[publishingv1.SchedulePostRequest],
) (*connect.Response[publishingv1.SchedulePostResponse], error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.posted = append(p.posted, req.Msg)
	return connect.NewResponse(&publishingv1.SchedulePostResponse{Outcome: &publishingv1.SchedulePostResponse_Scheduled{
		Scheduled: &publishingv1.Scheduled{PostId: fmt.Sprintf("post-%d", len(p.posted))},
	}}), nil
}

// A battle starting in, with its cards.
func battle(id string, in time.Duration, cards int) *promov1.Teaser {
	starts := time.Now().UTC().Add(in).Truncate(time.Minute)
	teaser := &promov1.Teaser{
		BattleId: id, StartsAt: timestamppb.New(starts), EndsAt: timestamppb.New(starts.Add(time.Hour)), Times: 3,
		Description: "Close a ring around any land.",
	}
	for i := range cards {
		teaser.Cards = append(teaser.Cards, &promov1.TeaserCard{
			BackgroundUrl: "https://media.test/bg.jpg", Headline: fmt.Sprintf("Rule %d", i+1),
		})
	}
	return teaser
}

// The renderer's output for a teaser.json, in its folder.
func render(t *testing.T, folder string) {
	t.Helper()
	written, err := os.ReadFile(filepath.Join(folder, "teaser.json")) //nolint:gosec // a file the test made.
	require.NoError(t, err)
	var teaser promov1.Teaser
	require.NoError(t, protojson.Unmarshal(written, &teaser))
	require.NoError(t, os.WriteFile(filepath.Join(folder, "teaser.mp4"), []byte("mp4"), 0o600))
	for i := range teaser.GetCards() {
		require.NoError(t, os.WriteFile(filepath.Join(folder, fmt.Sprintf("card-%d.jpg", i+1)), []byte("jpg"), 0o600))
	}
}

func TestTheBattlesDueAreWrittenForTheRendererAndTheTeasedLeftOut(t *testing.T) {
	ring, finale, teased := battle("entry-7", 32*time.Hour, 2), battle("season-1/finale", 35*time.Hour, 1), battle("entry-6", 5*time.Hour, 1)
	finale.EndsSeason = true
	b := serve(t, teased, ring, finale)
	b.teased = []string{"entry-6"}
	folder := t.TempDir()

	due, err := promo.ListDue(t.Context(), folder, config, slog.New(slog.DiscardHandler))

	require.NoError(t, err)
	require.Len(t, due, 2)
	assert.Equal(t, []string{"entry-7", "season-1_finale"}, []string{due[0].Folder, due[1].Folder})
	written, err := os.ReadFile(filepath.Join(folder, "season-1_finale", "teaser.json")) //nolint:gosec // a file the test made.
	require.NoError(t, err)
	var kept promov1.Teaser
	require.NoError(t, protojson.Unmarshal(written, &kept))
	assert.Equal(t, "season-1/finale", kept.GetBattleId())
	assert.True(t, kept.GetEndsSeason())
	assert.NoDirExists(t, filepath.Join(folder, "entry-6"))
}

func TestARenderedTeaserGoesOutAsACarouselOnTikTokAndASlideshowOnYouTubeByItsBattle(t *testing.T) {
	ring := battle("season-1/finale", 32*time.Hour, 2)
	b := serve(t, ring)
	folder := t.TempDir()
	_, err := promo.ListDue(t.Context(), folder, config, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	render(t, filepath.Join(folder, "season-1_finale"))

	report, err := promo.PostTeasers(t.Context(), folder, config, slog.New(slog.DiscardHandler))

	require.NoError(t, err)
	require.Len(t, report, 1)
	starts := ring.GetStartsAt().AsTime()
	assert.Equal(t, promo.TeaserScheduled, report[0].Result)
	assert.Equal(t, starts.Add(-30*time.Hour), report[0].At)
	assert.Equal(t, []promo.TeaserPlatform{"tiktok", "youtube"}, report[0].Platforms)
	assert.False(t, report.Failed())

	require.Len(t, b.planned, 1)
	assert.Equal(t, starts.Add(-time.Hour), b.planned[0].GetBy().AsTime())
	require.Len(t, b.posted, 1)
	post := b.posted[0]
	assert.Equal(t, "https://media.example/teaser.mp4", post.GetMediaUrl())
	assert.Equal(t, []string{"https://media.example/card-1.jpg", "https://media.example/card-2.jpg"}, post.GetTiktokPhotoUrls())
	assert.NotEmpty(t, post.GetTiktok().GetTitle())
	assert.Equal(t, "season-1/finale", post.GetMetadata().GetFields()["teaserBattle"].GetStringValue())
	assert.ElementsMatch(t, []string{"/temp/teaser.mp4", "/temp/card-1.jpg", "/temp/card-2.jpg"}, b.uploads)

	// The next run finds it teased, and posts it no more.
	b.teased = []string{"season-1/finale"}
	report, err = promo.PostTeasers(t.Context(), folder, config, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	assert.Equal(t, promo.TeaserSkipped, report[0].Result)
	assert.Len(t, b.posted, 1)
}

func TestATeaserTikTokCannotTakeByItsBattleGoesOnYouTubeAloneWithNoPhoto(t *testing.T) {
	b := serve(t, battle("entry-7", 32*time.Hour, 2))
	b.open = []publishingv1.Platform{youtube}
	folder := t.TempDir()
	_, err := promo.ListDue(t.Context(), folder, config, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	render(t, filepath.Join(folder, "entry-7"))

	report, err := promo.PostTeasers(t.Context(), folder, config, slog.New(slog.DiscardHandler))

	require.NoError(t, err)
	assert.Equal(t, []promo.TeaserPlatform{"youtube"}, report[0].Platforms)
	assert.Len(t, b.planned, 3, "both, TikTok alone, then YouTube alone")
	assert.Empty(t, b.posted[0].GetTiktokPhotoUrls())
	assert.Equal(t, []string{"/temp/teaser.mp4"}, b.uploads)
}

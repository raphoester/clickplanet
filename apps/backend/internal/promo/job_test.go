package promo_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1/promov1connect"
	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1/publishingv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo"
)

var at = time.Date(2026, 11, 13, 14, 0, 0, 0, time.UTC)

// The backend as the job reaches it: promo's teasers and publishing's upload links, fakes on httptest, behind an ssh on
// the PATH that sends each call there with curl, and says a refusal as busybox's wget does.
type backend struct {
	promov1connect.UnimplementedAdminServiceHandler

	mu      sync.Mutex
	due     []*promov1.Teaser
	answer  *promov1.PostTeaserResponse
	refusal error
	posted  []*promov1.PostTeaserRequest
	uploads []string
	storage *httptest.Server
}

type uploads struct {
	publishingv1connect.UnimplementedAdminServiceHandler

	b *backend
}

func serve(t *testing.T, due ...*promov1.Teaser) *backend {
	t.Helper()

	b := &backend{due: due, answer: &promov1.PostTeaserResponse{Outcome: &promov1.PostTeaserResponse_Posted{Posted: &promov1.TeaserPosted{
		PostId: "post-1", Title: "BATTLE ON SATURDAY 14 NOVEMBER AT 20:00 UTC", At: timestamppb.New(at), Platforms: []string{"tiktok", "youtube"},
	}}}}
	b.storage = httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		b.mu.Lock()
		b.uploads = append(b.uploads, r.URL.Path)
		b.mu.Unlock()
	}))
	t.Cleanup(b.storage.Close)

	mux := http.NewServeMux()
	mux.Handle(promov1connect.NewAdminServiceHandler(b))
	mux.Handle(publishingv1connect.NewAdminServiceHandler(uploads{b: b}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	folder := t.TempDir()
	script := fmt.Sprintf(`#!/bin/sh
method=${3##*8081/}
body=$(mktemp)
code=$(curl -s -o "$body" -w '%%{http_code}' -X POST -H 'Content-Type: application/json' --data-binary @- %q/"$method")
if [ "$code" -ge 400 ]; then echo "wget: server returned error: HTTP/1.1 $code" >&2; exit 1; fi
cat "$body"
`, server.URL)
	require.NoError(t, os.WriteFile(filepath.Join(folder, "ssh"), []byte(script), 0o700)) //nolint:gosec // a script the test runs.
	t.Setenv("PATH", folder+string(os.PathListSeparator)+os.Getenv("PATH"))
	return b
}

var config = promo.JobConfig{SSH: "deploy@vps.example"}

func (b *backend) ListDueTeasers(context.Context, *connect.Request[promov1.ListDueTeasersRequest]) (*connect.Response[promov1.ListDueTeasersResponse], error) {
	return connect.NewResponse(&promov1.ListDueTeasersResponse{Teasers: b.due}), nil
}

func (b *backend) PostTeaser(_ context.Context, req *connect.Request[promov1.PostTeaserRequest]) (*connect.Response[promov1.PostTeaserResponse], error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.posted = append(b.posted, req.Msg)
	if b.refusal != nil {
		return nil, b.refusal
	}
	return connect.NewResponse(b.answer), nil
}

func (u uploads) CreateMediaUpload(
	_ context.Context, req *connect.Request[publishingv1.CreateMediaUploadRequest],
) (*connect.Response[publishingv1.CreateMediaUploadResponse], error) {
	return connect.NewResponse(&publishingv1.CreateMediaUploadResponse{
		UploadUrl: u.b.storage.URL + "/temp/" + req.Msg.GetFileName(), MediaUrl: "https://media.example/" + req.Msg.GetFileName(),
	}), nil
}

// A battle with its cards.
func battle(id string, cards int) *promov1.Teaser {
	teaser := &promov1.Teaser{BattleId: id, StartsAt: timestamppb.New(at.Add(30 * time.Hour)), Times: 3}
	for i := range cards {
		teaser.Cards = append(teaser.Cards, &promov1.TeaserCard{BackgroundUrl: "https://media.test/bg.jpg", Headline: fmt.Sprintf("Rule %d", i+1)})
	}
	return teaser
}

// The renderer's output for the teaser.json in folder.
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

func TestEachBattlePromoSaysIsDueIsWrittenForTheRenderer(t *testing.T) {
	finale := battle("season-1/finale", 1)
	finale.EndsSeason = true
	serve(t, battle("entry-7", 2), finale)
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
}

func TestEachRenderIsUploadedAndHandedToPromoWhichSaysWhatBecameOfIt(t *testing.T) {
	b := serve(t, battle("entry-7", 2))
	folder := t.TempDir()
	_, err := promo.ListDue(t.Context(), folder, config, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	render(t, filepath.Join(folder, "entry-7"))

	report, err := promo.PostTeasers(t.Context(), folder, config, slog.New(slog.DiscardHandler))

	require.NoError(t, err)
	require.Len(t, report, 1)
	assert.Equal(t, promo.TeaserPosted, report[0].Result)
	assert.Equal(t, at, report[0].At)
	assert.Equal(t, []string{"tiktok", "youtube"}, report[0].Platforms)
	assert.False(t, report.Failed())
	assert.ElementsMatch(t, []string{"/temp/card-1.jpg", "/temp/card-2.jpg", "/temp/teaser.mp4"}, b.uploads)
	require.Len(t, b.posted, 1)
	assert.Equal(t, "entry-7", b.posted[0].GetBattleId())
	assert.Equal(t, []string{"https://media.example/card-1.jpg", "https://media.example/card-2.jpg"}, b.posted[0].GetPhotoUrls())
	assert.Equal(t, "https://media.example/teaser.mp4", b.posted[0].GetVideoUrl())
}

func TestWhatPromoDidNotPostIsInTheReportAndARefusalFailsTheRun(t *testing.T) {
	b := serve(t, battle("entry-7", 1))
	folder := t.TempDir()
	_, err := promo.ListDue(t.Context(), folder, config, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	render(t, filepath.Join(folder, "entry-7"))

	b.answer = &promov1.PostTeaserResponse{Outcome: &promov1.PostTeaserResponse_Skipped{Skipped: &promov1.TeaserNotPosted{Reason: "teased already"}}}
	report, err := promo.PostTeasers(t.Context(), folder, config, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	assert.Equal(t, promo.TeaserSkipped, report[0].Result)
	assert.Equal(t, "teased already", report[0].Why)
	assert.False(t, report.Failed())

	b.refusal = connect.NewError(connect.CodeInvalidArgument, errors.New("2 photos for 1 cards"))
	report, err = promo.PostTeasers(t.Context(), folder, config, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	assert.Contains(t, report[0].Why, "the backend refused promo.v1.AdminService/PostTeaser")
	assert.True(t, report.Failed(), "a refusal is never waited for")
}

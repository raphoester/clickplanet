package ssh_backend_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers/ssh_backend"
)

const host = "deploy@clickplanet.lol"

var (
	start   = time.Date(2026, 11, 14, 20, 0, 0, 0, time.UTC)
	at      = time.Date(2026, 11, 13, 14, 0, 0, 0, time.UTC)
	both    = []teasers.Platform{teasers.TikTok, teasers.YouTube}
	youtube = []teasers.Platform{teasers.YouTube}
	wired   = []publishingv1.Platform{publishingv1.Platform_PLATFORM_TIKTOK, publishingv1.Platform_PLATFORM_YOUTUBE}
)

type fakeSSH struct {
	client string
	folder string
}

// An ssh that keeps its arguments and its standard input, then says what call n is told to (says-n), and exits as told.
func ssh(t *testing.T, says []string, complains string, exit int) fakeSSH {
	t.Helper()
	folder := t.TempDir()
	for i, said := range says {
		require.NoError(t, os.WriteFile(filepath.Join(folder, fmt.Sprintf("says-%d", i+1)), []byte(said), 0o600))
	}
	fake := fakeSSH{client: filepath.Join(folder, "ssh"), folder: folder}
	script := fmt.Sprintf(`#!/bin/sh
n=$(( $(ls %[1]q | grep -c '^body-') + 1 ))
for arg in "$@"; do printf '%%s\n' "$arg"; done > %[1]q/args-$n
cat > %[1]q/body-$n
cat %[1]q/says-$n 2> /dev/null
printf '%%s' %[2]q >&2
exit %[3]d
`, folder, complains, exit)
	require.NoError(t, os.WriteFile(fake.client, []byte(script), 0o700)) //nolint:gosec // a script the test runs.
	return fake
}

// Call n's service and method, its whole command checked, and the request it sent.
func (f fakeSSH) call(t *testing.T, n int, request proto.Message) string {
	t.Helper()
	args, err := os.ReadFile(filepath.Join(f.folder, fmt.Sprintf("args-%d", n)))
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSuffix(string(args), "\n"), "\n")
	require.Len(t, lines, 3)
	assert.Equal(t, []string{"--", host}, lines[:2])
	method, found := strings.CutPrefix(lines[2], "cd /opt/clickplanet/deploy/vps && docker compose exec -T backend wget -qO-"+
		" --header 'Content-Type: application/json' --post-file=/dev/stdin http://127.0.0.1:8081/")
	require.True(t, found, lines[2])
	body, err := os.ReadFile(filepath.Join(f.folder, fmt.Sprintf("body-%d", n)))
	require.NoError(t, err)
	require.NoError(t, protojson.Unmarshal(body, request))
	return method
}

func (f fakeSSH) calls(t *testing.T) int {
	t.Helper()
	bodies, err := filepath.Glob(filepath.Join(f.folder, "body-*"))
	require.NoError(t, err)
	return len(bodies)
}

func backend(t *testing.T, fake fakeSSH, storage *http.Client) *ssh_backend.Backend {
	t.Helper()
	if storage == nil {
		storage = http.DefaultClient
	}
	backend, err := ssh_backend.New(host, fake.client, storage)
	require.NoError(t, err)
	return backend
}

func written(t *testing.T, message proto.Message) string {
	t.Helper()
	body, err := protojson.Marshal(message)
	require.NoError(t, err)
	return string(body)
}

func ringWar() teasers.Teaser {
	return teasers.Teaser{
		Battle: "entry-7", StartsAt: start, EndsAt: start.Add(time.Hour), Description: "Close a ring around any land.", Times: 3,
		Cards: []teasers.Card{{BackgroundURL: "https://media.test/ring.jpg", Headline: "Circle any land", Line: "Every click."}},
	}
}

func TestTheBattlesToComeAreAskedOfPromoWithTheirCards(t *testing.T) {
	fake := ssh(t, []string{written(t, &promov1.ListTeasersResponse{Teasers: []*promov1.Teaser{{
		BattleId: "entry-7", StartsAt: timestamppb.New(start), EndsAt: timestamppb.New(start.Add(time.Hour)),
		Description: "Close a ring around any land.", Times: 3,
		Cards: []*promov1.TeaserCard{{BackgroundUrl: "https://media.test/ring.jpg", Headline: "Circle any land", Line: "Every click."}},
	}, {BattleId: "season-1/finale", StartsAt: timestamppb.New(start.Add(24 * time.Hour)), EndsSeason: true}}})}, "", 0)

	listed, err := backend(t, fake, nil).Teasers(t.Context(), 36*time.Hour)

	require.NoError(t, err)
	finale := teasers.Teaser{Battle: "season-1/finale", StartsAt: start.Add(24 * time.Hour), EndsAt: time.Unix(0, 0).UTC(), EndsSeason: true}
	assert.Equal(t, []teasers.Teaser{ringWar(), finale}, listed)
	var asked promov1.ListTeasersRequest
	assert.Equal(t, "promo.v1.AdminService/ListTeasers", fake.call(t, 1, &asked))
	assert.Equal(t, 36*time.Hour, asked.GetWithin().AsDuration())
}

func TestTheBattlesTeasedAreThoseThePostsMetadataNames(t *testing.T) {
	fake := ssh(t, []string{`{"posts":[` +
		`{"postId":"p-1","title":"BATTLE","status":"scheduled","metadata":{"teaserBattle":"entry-7","teaserStartsAt":"2026-11-14T20:00:00Z"}},` +
		`{"postId":"p-2","title":"FRANCE VS TURKEY","status":"published","metadata":{"storyFlag":"fr"}},` +
		`{"postId":"p-3","title":"BY HAND","status":"published"},` +
		`{"postId":"p-4","title":"FINAL BATTLE","status":"cancelled","metadata":{"teaserBattle":"season-1/finale"}},` +
		`{"postId":"p-5","title":"BATTLE","status":"failed","metadata":{"teaserBattle":"entry-7"}}` +
		`]}`}, "", 0)
	since := time.Date(2026, 11, 11, 7, 43, 0, 0, time.UTC)

	teased, err := backend(t, fake, nil).Teased(t.Context(), since)

	require.NoError(t, err)
	assert.Equal(t, []teasers.BattleID{"entry-7", "season-1/finale"}, teased)
	var asked publishingv1.ListPostsRequest
	assert.Equal(t, "publishing.v1.AdminService/ListPosts", fake.call(t, 1, &asked))
	assert.Equal(t, since, asked.GetSince().AsTime())
}

func TestAReadTheBackendRefusesIsAnErrorThatSaysWhy(t *testing.T) {
	fake := ssh(t, nil, "wget: server returned error: HTTP/1.1 412 Precondition Failed", 1)

	_, err := backend(t, fake, nil).Teased(t.Context(), at)
	require.ErrorContains(t, err, "ssh failed to ask publishing.v1.AdminService/ListPosts")
	require.ErrorContains(t, err, "412 Precondition Failed")

	_, err = backend(t, fake, nil).Teasers(t.Context(), time.Hour)
	require.ErrorContains(t, err, "ssh failed to ask promo.v1.AdminService/ListTeasers")
}

func TestASlotIsTheCalendarsByTheBattleOrNone(t *testing.T) {
	fake := ssh(t, []string{written(t, &publishingv1.PlanPostResponse{Slot: &publishingv1.Slot{At: timestamppb.New(at), Platforms: wired}}), `{}`}, "", 0)
	publishing := backend(t, fake, nil)
	notBefore, by := ringWar().Window()

	slot, found, err := publishing.Slot(t.Context(), both, notBefore, by)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, teasers.Slot{At: at, Platforms: both}, slot)
	var asked publishingv1.PlanPostRequest
	assert.Equal(t, "publishing.v1.AdminService/PlanPost", fake.call(t, 1, &asked))
	assert.Equal(t, wired, asked.GetPlatforms())
	assert.Equal(t, notBefore, asked.GetNotBefore().AsTime())
	assert.Equal(t, by, asked.GetBy().AsTime())

	_, found, err = publishing.Slot(t.Context(), both, notBefore, by)
	require.NoError(t, err)
	assert.False(t, found)
}

func TestASlotTheBackendCannotBeAskedIsBusy(t *testing.T) {
	_, _, err := backend(t, ssh(t, nil, "connection refused", 255), nil).Slot(t.Context(), both, at, start)

	var busy *teasers.BusyError
	require.ErrorAs(t, err, &busy)
	assert.Contains(t, busy.Reason, "connection refused")
}

// A storage that keeps what was put at each path.
type storage struct {
	mu   sync.Mutex
	puts map[string]string
}

func serve(t *testing.T, status int) (*storage, *httptest.Server) {
	t.Helper()
	kept := &storage{puts: map[string]string{}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		kept.mu.Lock()
		kept.puts[r.URL.Path] = r.Header.Get("Content-Type") + " " + string(body)
		kept.mu.Unlock()
		w.WriteHeader(status)
	}))
	t.Cleanup(server.Close)
	return kept, server
}

// The renderer's files, each holding its own name.
func rendered(t *testing.T) teasers.Media {
	t.Helper()
	folder := t.TempDir()
	media := teasers.Media{Video: teasers.File(filepath.Join(folder, "teaser.mp4"))}
	for _, name := range []string{"card-1.jpg", "card-2.jpg"} {
		media.Photos = append(media.Photos, teasers.File(filepath.Join(folder, name)))
	}
	for _, file := range append([]teasers.File{media.Video}, media.Photos...) {
		require.NoError(t, os.WriteFile(string(file), []byte(filepath.Base(string(file))), 0o600))
	}
	return media
}

func link(t *testing.T, server *httptest.Server, name string) string {
	t.Helper()
	return written(t, &publishingv1.CreateMediaUploadResponse{
		UploadUrl: server.URL + "/upload/" + name, MediaUrl: "https://media.example/" + name,
	})
}

func TestATeaserOnTikTokGoesWithItsPhotosAndItsVideoForYouTube(t *testing.T) {
	kept, server := serve(t, http.StatusOK)
	fake := ssh(t, []string{
		link(t, server, "teaser.mp4"), link(t, server, "card-1.jpg"), link(t, server, "card-2.jpg"),
		written(t, &publishingv1.SchedulePostResponse{Outcome: &publishingv1.SchedulePostResponse_Scheduled{
			Scheduled: &publishingv1.Scheduled{PostId: "post-1", Refusals: []*publishingv1.Refusal{
				{Platform: publishingv1.Platform_PLATFORM_YOUTUBE, Reason: "no YouTube account is connected"},
			}},
		}}),
	}, "", 0)
	post := teasers.Post{Teaser: ringWar(), Media: rendered(t), Slot: teasers.Slot{At: at, Platforms: both}}

	receipt, err := backend(t, fake, server.Client()).Schedule(t.Context(), post)

	require.NoError(t, err)
	assert.Equal(t, teasers.Receipt{ID: "post-1", Refused: map[teasers.Platform]string{teasers.YouTube: "no YouTube account is connected"}}, receipt)
	for n, upload := range []*publishingv1.CreateMediaUploadRequest{
		{FileName: "teaser.mp4", ContentType: "video/mp4", SizeBytes: 10},
		{FileName: "card-1.jpg", ContentType: "image/jpeg", SizeBytes: 10},
		{FileName: "card-2.jpg", ContentType: "image/jpeg", SizeBytes: 10},
	} {
		var asked publishingv1.CreateMediaUploadRequest
		assert.Equal(t, "publishing.v1.AdminService/CreateMediaUpload", fake.call(t, n+1, &asked))
		assert.True(t, proto.Equal(upload, &asked), "%v", &asked)
	}
	assert.Equal(t, map[string]string{
		"/upload/teaser.mp4": "video/mp4 teaser.mp4", "/upload/card-1.jpg": "image/jpeg card-1.jpg", "/upload/card-2.jpg": "image/jpeg card-2.jpg",
	}, kept.puts)

	var asked publishingv1.SchedulePostRequest
	assert.Equal(t, "publishing.v1.AdminService/SchedulePost", fake.call(t, 4, &asked))
	words := ringWar().Words()
	assert.Equal(t, written(t, &publishingv1.SchedulePostRequest{
		Slot:     &publishingv1.Slot{At: timestamppb.New(at), Platforms: wired},
		MediaUrl: "https://media.example/teaser.mp4", IdempotencyKey: post.Key(), Title: words.Title,
		Tiktok: &publishingv1.TikTokWords{Caption: words.TikTok.Caption, Title: words.TikTok.Title},
		Youtube: &publishingv1.YouTubeWords{
			Title: words.YouTube.Title, Description: words.YouTube.Description, Tags: words.YouTube.Tags,
		},
		TiktokPhotoUrls: []string{"https://media.example/card-1.jpg", "https://media.example/card-2.jpg"},
	}), written(t, func() *publishingv1.SchedulePostRequest { asked.Metadata = nil; return &asked }()))
	assert.Equal(t, "entry-7", fake.metadata(t, 4)["teaserBattle"])
}

// The metadata call n sent.
func (f fakeSSH) metadata(t *testing.T, n int) map[string]any {
	t.Helper()
	var asked publishingv1.SchedulePostRequest
	f.call(t, n, &asked)
	return asked.GetMetadata().AsMap()
}

func TestATeaserOnYouTubeAloneUploadsNoPhoto(t *testing.T) {
	_, server := serve(t, http.StatusOK)
	fake := ssh(t, []string{
		link(t, server, "teaser.mp4"),
		written(t, &publishingv1.SchedulePostResponse{Outcome: &publishingv1.SchedulePostResponse_Scheduled{Scheduled: &publishingv1.Scheduled{PostId: "post-2"}}}),
	}, "", 0)

	_, err := backend(t, fake, server.Client()).Schedule(t.Context(), teasers.Post{Teaser: ringWar(), Media: rendered(t), Slot: teasers.Slot{At: at, Platforms: youtube}})

	require.NoError(t, err)
	assert.Equal(t, 2, fake.calls(t))
	var asked publishingv1.SchedulePostRequest
	fake.call(t, 2, &asked)
	assert.Empty(t, asked.GetTiktokPhotoUrls())
	assert.Equal(t, []publishingv1.Platform{publishingv1.Platform_PLATFORM_YOUTUBE}, asked.GetSlot().GetPlatforms())
}

func TestEachFileIsUploadedOnceHoweverManyTimesItsPostIsTried(t *testing.T) {
	_, server := serve(t, http.StatusOK)
	busy := written(t, &publishingv1.SchedulePostResponse{Outcome: &publishingv1.SchedulePostResponse_Busy{
		Busy: &publishingv1.Busy{Reason: "zernio answered 429", RetryAfter: durationpb.New(time.Minute)},
	}})
	fake := ssh(t, []string{link(t, server, "teaser.mp4"), link(t, server, "card-1.jpg"), link(t, server, "card-2.jpg"), busy, busy}, "", 0)
	publishing := backend(t, fake, server.Client())
	post := teasers.Post{Teaser: ringWar(), Media: rendered(t), Slot: teasers.Slot{At: at, Platforms: both}}

	for range 2 {
		_, err := publishing.Schedule(t.Context(), post)
		var refusal *teasers.BusyError
		require.ErrorAs(t, err, &refusal)
		assert.Equal(t, &teasers.BusyError{RetryAfter: time.Minute, Reason: "zernio answered 429"}, refusal)
	}
	assert.Equal(t, 5, fake.calls(t))
}

func TestWhatTheModuleAnswersOfAPostIsItsOutcome(t *testing.T) {
	_, server := serve(t, http.StatusOK)
	for name, answer := range map[string]struct {
		said  *publishingv1.SchedulePostResponse
		check func(t *testing.T, err error)
	}{
		"taken": {
			said: &publishingv1.SchedulePostResponse{Outcome: &publishingv1.SchedulePostResponse_Taken{Taken: &publishingv1.Taken{Reason: "the slot is no longer open"}}},
			check: func(t *testing.T, err error) {
				t.Helper()
				require.ErrorIs(t, err, teasers.ErrTaken)
			},
		},
		"failed": {
			said: &publishingv1.SchedulePostResponse{Outcome: &publishingv1.SchedulePostResponse_Failed{Failed: &publishingv1.Failed{Reason: "the caption is refused"}}},
			check: func(t *testing.T, err error) {
				t.Helper()
				require.EqualError(t, err, "the caption is refused")
				var busy *teasers.BusyError
				assert.NotErrorAs(t, err, &busy)
			},
		},
		"nothing": {
			said: &publishingv1.SchedulePostResponse{},
			check: func(t *testing.T, err error) {
				t.Helper()
				require.EqualError(t, err, "the backend said nothing of the post")
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			fake := ssh(t, []string{link(t, server, "teaser.mp4"), written(t, answer.said)}, "", 0)
			_, err := backend(t, fake, server.Client()).Schedule(t.Context(), teasers.Post{
				Teaser: ringWar(), Media: rendered(t), Slot: teasers.Slot{At: at, Platforms: youtube},
			})
			answer.check(t, err)
		})
	}
}

func TestAnUploadTheStorageRefusesSaysSoWithoutItsAddress(t *testing.T) {
	_, refusing := serve(t, http.StatusForbidden)
	fake := ssh(t, []string{link(t, refusing, "teaser.mp4")}, "", 0)

	_, err := backend(t, fake, refusing.Client()).Schedule(t.Context(), teasers.Post{Teaser: ringWar(), Media: rendered(t), Slot: teasers.Slot{At: at, Platforms: youtube}})

	require.EqualError(t, err, "the storage answered 403 to the upload of a file")
	assert.NotContains(t, err.Error(), refusing.URL)

	_, busy := serve(t, http.StatusServiceUnavailable)
	fake = ssh(t, []string{link(t, busy, "teaser.mp4")}, "", 0)
	_, err = backend(t, fake, busy.Client()).Schedule(t.Context(), teasers.Post{Teaser: ringWar(), Media: rendered(t), Slot: teasers.Slot{At: at, Platforms: youtube}})
	var refusal *teasers.BusyError
	require.ErrorAs(t, err, &refusal)
}

package ssh_publishing_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/ssh_publishing"
)

const host = "deploy@clickplanet.lol"

var at = time.Date(2026, 10, 12, 16, 40, 0, 0, time.UTC)

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

// Call n's method, its whole command checked, and the request it sent.
func (f fakeSSH) call(t *testing.T, n int, request proto.Message) string {
	t.Helper()

	args, err := os.ReadFile(filepath.Join(f.folder, fmt.Sprintf("args-%d", n)))
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSuffix(string(args), "\n"), "\n")
	require.Len(t, lines, 3)
	assert.Equal(t, []string{"--", host}, lines[:2])
	method, found := strings.CutPrefix(lines[2], "cd /opt/clickplanet/deploy/vps && docker compose exec -T backend wget -qO-"+
		" --header 'Content-Type: application/json' --post-file=/dev/stdin http://127.0.0.1:8081/publishing.v1.AdminService/")
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

func service(t *testing.T, fake fakeSSH, storage *http.Client) *ssh_publishing.Service {
	t.Helper()

	if storage == nil {
		storage = http.DefaultClient
	}
	service, err := ssh_publishing.New(host, fake.client, storage)
	require.NoError(t, err)
	return service
}

func written(t *testing.T, message proto.Message) string {
	t.Helper()

	body, err := protojson.Marshal(message)
	require.NoError(t, err)
	return string(body)
}

func TestTheRoomOfEachPlatformIsAskedOfTheBackend(t *testing.T) {
	fake := ssh(t, []string{written(t, &publishingv1.GetRoomResponse{Rooms: []*publishingv1.PlatformRoom{
		{Platform: publishingv1.Platform_PLATFORM_TIKTOK, Left: 1}, {Platform: publishingv1.Platform_PLATFORM_YOUTUBE, Left: 9},
	}})}, "", 0)

	room, err := service(t, fake, nil).Room(t.Context(), []posts.Platform{posts.TikTok, posts.YouTube})

	require.NoError(t, err)
	assert.Equal(t, map[posts.Platform]int{posts.TikTok: 1, posts.YouTube: 9}, room)
	var asked publishingv1.GetRoomRequest
	assert.Equal(t, "GetRoom", fake.call(t, 1, &asked))
	assert.Equal(t, []publishingv1.Platform{publishingv1.Platform_PLATFORM_TIKTOK, publishingv1.Platform_PLATFORM_YOUTUBE}, asked.GetPlatforms())
}

func TestTheStoriesToldAreThoseOfThePostsWithAKeyAndATime(t *testing.T) {
	fake := ssh(t, []string{`{"posts":[` +
		`{"postId":"p-1","title":"ALGERIA IS INVADING LIBYA","status":"scheduled","scheduledFor":"2026-10-14T05:15:00Z",` +
		`"metadata":{"storyFlag":"dz","storyContinent":"Africa","storyCountries":["ly","dz"],"usageCounted":true}},` +
		`{"postId":"p-2","status":"published","scheduledFor":"2026-10-13T20:00:00Z"},` +
		`{"postId":"p-3","title":"A DRAFT","status":"draft","metadata":{"storyFlag":"pt","storyContinent":"Europe"}},` +
		`{"postId":"p-4","title":"FRANCE WINS DAY 5","status":"failed","scheduledFor":"2026-10-14T07:15:00Z",` +
		`"failure":"tiktok: TikTok creator has reached the daily posting limit.",` +
		`"metadata":{"storyFlag":"fr","storyContinent":"the world","storyAngle":"round","storyRound":"1.5"},"addedLater":1}` +
		`]}`}, "", 0)
	since := time.Date(2026, 10, 13, 0, 57, 0, 0, time.UTC)

	told, err := service(t, fake, nil).Told(t.Context(), since)

	require.NoError(t, err)
	assert.Equal(t, []posts.Telling{
		{
			Key: posts.StoryKey{Flag: "dz", Continent: "Africa"}, Countries: []string{"ly", "dz"},
			At: time.Date(2026, 10, 14, 5, 15, 0, 0, time.UTC), Title: "ALGERIA IS INVADING LIBYA", State: "scheduled",
		},
		{
			Key: posts.StoryKey{Flag: "fr", Continent: "the world", Angle: posts.DayWon, Round: "1.5"},
			At:  time.Date(2026, 10, 14, 7, 15, 0, 0, time.UTC), Title: "FRANCE WINS DAY 5", State: "failed",
			Failure: "tiktok: TikTok creator has reached the daily posting limit.",
		},
	}, told)
	var asked publishingv1.ListPostsRequest
	assert.Equal(t, "ListPosts", fake.call(t, 1, &asked))
	assert.Equal(t, since, asked.GetSince().AsTime())
}

func TestAHistoryTheBackendDoesNotGiveIsAnErrorThatSaysWhy(t *testing.T) {
	fake := ssh(t, nil, "wget: server returned error: HTTP/1.1 400 Bad Request", 1)

	_, err := service(t, fake, nil).Told(t.Context(), at)

	require.ErrorContains(t, err, "ssh failed to ask ListPosts")
	require.ErrorContains(t, err, "400 Bad Request")
}

func TestASlotIsTheCalendarsOrNone(t *testing.T) {
	fake := ssh(t, []string{
		written(t, &publishingv1.PlanPostResponse{Slot: &publishingv1.Slot{
			At: timestamppb.New(at), Platforms: []publishingv1.Platform{publishingv1.Platform_PLATFORM_YOUTUBE},
		}}),
		`{}`,
	}, "", 0)
	publishing := service(t, fake, nil)

	slot, found, err := publishing.Slot(t.Context(), []posts.Platform{posts.TikTok, posts.YouTube})
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, posts.Slot{At: at, Platforms: []posts.Platform{posts.YouTube}}, slot)
	var asked publishingv1.PlanPostRequest
	assert.Equal(t, "PlanPost", fake.call(t, 1, &asked))
	assert.Nil(t, asked.GetNotBefore())

	_, found, err = publishing.Slot(t.Context(), []posts.Platform{posts.TikTok})
	require.NoError(t, err)
	assert.False(t, found)
}

func TestAnSSHThatFailsToPlanIsBusySinceTheBackendMayBeRestarting(t *testing.T) {
	fake := ssh(t, nil, "Connection refused", 255)

	_, _, err := service(t, fake, nil).Slot(t.Context(), []posts.Platform{posts.TikTok})

	var busy *posts.BusyError
	require.ErrorAs(t, err, &busy)
	assert.Contains(t, busy.Reason, "Connection refused")
}

// The storage of the posting service, as an upload link names it: what it was sent, and the status it answers.
type storage struct {
	status  int
	sent    []byte
	headers http.Header
	puts    int
}

func (s *storage) serve(t *testing.T) (string, *http.Client) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		s.puts++
		s.headers = r.Header.Clone()
		s.sent, _ = io.ReadAll(r.Body)
		w.WriteHeader(s.status)
	}))
	t.Cleanup(server.Close)
	return server.URL + "/temp/clip-1.mp4?signature=the-signature", server.Client()
}

func postOf(t *testing.T) posts.Post {
	t.Helper()

	video := filepath.Join(t.TempDir(), "clip-1.mp4")
	require.NoError(t, os.WriteFile(video, []byte("the video"), 0o600))
	clip, err := posts.ClipSnapshot{
		Pick: 1, Title: "FRANCE IS ATTACKING EUROPE", Teasers: []string{"FRANCE TAKES OVER EUROPE"},
		Countries: []posts.TaggedCountry{{Code: "fr", Tag: "france"}}, ContinentTag: "europe",
		Key: posts.StoryKey{Flag: "fr", Continent: "Europe"}, Video: posts.VideoFile(video),
	}.Restore()
	require.NoError(t, err)
	both := []posts.Platform{posts.TikTok, posts.YouTube}
	return posts.Plan{Platforms: both}.Post(clip, 1, 1, posts.LoadedDice(0)).At(posts.Slot{At: at, Platforms: both})
}

func uploadLink(address string) string {
	return fmt.Sprintf(`{"uploadUrl":%q,"mediaUrl":"https://media.example/clip-1.mp4"}`, address)
}

func TestAPostIsItsVideoUploadedToTheStorageThenItsWordsAndWhatItKeepsAtItsSlot(t *testing.T) {
	store := &storage{status: http.StatusOK}
	address, client := store.serve(t)
	fake := ssh(t, []string{uploadLink(address), written(t, &publishingv1.SchedulePostResponse{
		Outcome: &publishingv1.SchedulePostResponse_Scheduled{Scheduled: &publishingv1.Scheduled{
			PostId: "post-1", Refusals: []*publishingv1.Refusal{{Platform: publishingv1.Platform_PLATFORM_YOUTUBE, Reason: "no youtube account"}},
		}},
	})}, "", 0)
	post := postOf(t)

	scheduled, err := service(t, fake, client).Schedule(t.Context(), post)

	require.NoError(t, err)
	assert.Equal(t, posts.PostID("post-1"), scheduled.ID)
	require.ErrorContains(t, scheduled.Refused[posts.YouTube], "no youtube account")
	assert.Len(t, scheduled.Refused, 1)

	var link publishingv1.CreateMediaUploadRequest
	assert.Equal(t, "CreateMediaUpload", fake.call(t, 1, &link))
	assert.True(t, proto.Equal(&publishingv1.CreateMediaUploadRequest{FileName: "clip-1.mp4", ContentType: "video/mp4", SizeBytes: 9}, &link))
	assert.Equal(t, []byte("the video"), store.sent)
	assert.Equal(t, "video/mp4", store.headers.Get("Content-Type"))
	assert.Empty(t, store.headers.Get("Authorization"), "the storage takes no key")

	var request publishingv1.SchedulePostRequest
	assert.Equal(t, "SchedulePost", fake.call(t, 2, &request))
	snapshot := post.TakeSnapshot()
	key := sha256.Sum256([]byte("the video"))
	assert.Equal(t, hex.EncodeToString(key[:]), request.GetIdempotencyKey())
	assert.Equal(t, "https://media.example/clip-1.mp4", request.GetMediaUrl())
	assert.Equal(t, at, request.GetSlot().GetAt().AsTime())
	assert.Equal(t, []publishingv1.Platform{publishingv1.Platform_PLATFORM_TIKTOK, publishingv1.Platform_PLATFORM_YOUTUBE}, request.GetSlot().GetPlatforms())
	assert.Equal(t, "FRANCE IS ATTACKING EUROPE", request.GetTitle())
	assert.Equal(t, snapshot.TikTok.Caption, request.GetTiktok().GetCaption())
	assert.True(t, proto.Equal(&publishingv1.YouTubeWords{
		Title: snapshot.YouTube.Title, Description: snapshot.YouTube.Description, Tags: snapshot.YouTube.Tags,
	}, request.GetYoutube()))
	kept := request.GetMetadata().AsMap()
	assert.Equal(t, "fr", kept["storyFlag"])
	assert.Equal(t, []any{"fr"}, kept["storyCountries"])
	assert.Equal(t, "2026-10-12T16:40:00Z", kept["record"].(map[string]any)["post"].(map[string]any)["scheduledAt"], "the time it is planned at")
}

func TestAVideoIsUploadedOnceHoweverManyTimesItsPostIsTried(t *testing.T) {
	store := &storage{status: http.StatusOK}
	address, client := store.serve(t)
	busy := written(t, &publishingv1.SchedulePostResponse{Outcome: &publishingv1.SchedulePostResponse_Busy{Busy: &publishingv1.Busy{Reason: "busy"}}})
	fake := ssh(t, []string{uploadLink(address), busy, busy}, "", 0)
	publishing := service(t, fake, client)

	_, err := publishing.Schedule(t.Context(), postOf(t))
	require.Error(t, err)
	_, err = publishing.Schedule(t.Context(), postOf(t))
	require.Error(t, err)

	assert.Equal(t, 1, store.puts)
	assert.Equal(t, 3, fake.calls(t), "one link, two tries")
}

func TestEachOutcomeOfThePostIsWhatTheSenderActsOn(t *testing.T) {
	for name, outcome := range map[string]struct {
		answer *publishingv1.SchedulePostResponse
		check  func(t *testing.T, err error)
	}{
		"busy": {
			answer: &publishingv1.SchedulePostResponse{Outcome: &publishingv1.SchedulePostResponse_Busy{Busy: &publishingv1.Busy{
				RetryAfter: durationpb.New(90 * time.Second), Reason: "zernio answered 429 to the post: Rate limit exceeded",
			}}},
			check: func(t *testing.T, err error) {
				t.Helper()
				var busy *posts.BusyError
				require.ErrorAs(t, err, &busy)
				assert.Equal(t, 90*time.Second, busy.RetryAfter)
			},
		},
		"taken": {
			answer: &publishingv1.SchedulePostResponse{Outcome: &publishingv1.SchedulePostResponse_Taken{Taken: &publishingv1.Taken{Reason: "no longer open"}}},
			check: func(t *testing.T, err error) {
				t.Helper()
				require.ErrorIs(t, err, posts.ErrTaken)
			},
		},
		"failed": {
			answer: &publishingv1.SchedulePostResponse{Outcome: &publishingv1.SchedulePostResponse_Failed{Failed: &publishingv1.Failed{
				Reason: "zernio answered 400 to the post: caption too long",
			}}},
			check: func(t *testing.T, err error) {
				t.Helper()
				var busy *posts.BusyError
				require.ErrorContains(t, err, "caption too long")
				assert.NotErrorAs(t, err, &busy)
				assert.NotErrorIs(t, err, posts.ErrTaken)
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			store := &storage{status: http.StatusOK}
			address, client := store.serve(t)
			fake := ssh(t, []string{uploadLink(address), written(t, outcome.answer)}, "", 0)

			_, err := service(t, fake, client).Schedule(t.Context(), postOf(t))

			outcome.check(t, err)
		})
	}
}

func TestAStorageThatIsBusyOrCannotBeReachedIsBusyAndAnotherRefusalIsNotAndNeitherSaysTheAddress(t *testing.T) {
	for status, isBusy := range map[int]bool{http.StatusServiceUnavailable: true, http.StatusForbidden: false} {
		store := &storage{status: status}
		address, client := store.serve(t)
		fake := ssh(t, []string{uploadLink(address)}, "", 0)

		_, err := service(t, fake, client).Schedule(t.Context(), postOf(t))

		var busy *posts.BusyError
		require.ErrorContains(t, err, fmt.Sprintf("the storage answered %d", status))
		assert.Equal(t, isBusy, errors.As(err, &busy), "%d", status)
		assert.NotContains(t, err.Error(), "the-signature")
		assert.Equal(t, 1, fake.calls(t), "nothing posted")
	}

	fake := ssh(t, []string{uploadLink("http://127.0.0.1:1/temp/clip-1.mp4?signature=the-signature")}, "", 0)
	_, err := service(t, fake, nil).Schedule(t.Context(), postOf(t))
	var busy *posts.BusyError
	require.ErrorAs(t, err, &busy)
	assert.NotContains(t, err.Error(), "the-signature")
	assert.NotContains(t, err.Error(), "/temp/clip-1.mp4")
}

func TestAnEmptyHostIsRefused(t *testing.T) {
	_, err := ssh_publishing.New("", ssh_publishing.OpenSSH, http.DefaultClient)

	require.ErrorContains(t, err, "the ssh host is empty")
}

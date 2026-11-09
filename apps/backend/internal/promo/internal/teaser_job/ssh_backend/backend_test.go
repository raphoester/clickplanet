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
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teaser_job"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teaser_job/ssh_backend"
)

const host = "deploy@clickplanet.lol"

var (
	start = time.Date(2026, 11, 14, 20, 0, 0, 0, time.UTC)
	at    = time.Date(2026, 11, 13, 14, 0, 0, 0, time.UTC)
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
	args, err := os.ReadFile(filepath.Join(f.folder, fmt.Sprintf("args-%d", n))) //nolint:gosec // a file the test made.
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSuffix(string(args), "\n"), "\n")
	require.Len(t, lines, 3)
	assert.Equal(t, []string{"--", host}, lines[:2])
	method, found := strings.CutPrefix(lines[2], "cd /opt/clickplanet/deploy/vps && docker compose exec -T backend wget -qO-"+
		" --header 'Content-Type: application/json' --post-file=/dev/stdin http://127.0.0.1:8081/")
	require.True(t, found, lines[2])
	body, err := os.ReadFile(filepath.Join(f.folder, fmt.Sprintf("body-%d", n))) //nolint:gosec // a file the test made.
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
	made, err := ssh_backend.New(host, fake.client, storage)
	require.NoError(t, err)
	return made
}

func written(t *testing.T, message proto.Message) string {
	t.Helper()
	body, err := protojson.Marshal(message)
	require.NoError(t, err)
	return string(body)
}

func TestTheBattlesDueAreAskedOfPromoWithTheirTeaserAsWritten(t *testing.T) {
	finale := &promov1.Teaser{
		BattleId: "season-1/finale", StartsAt: timestamppb.New(start), EndsSeason: true,
		Cards: []*promov1.TeaserCard{{Headline: "Gain the most land"}, {Headline: "Bombs every 10 min"}},
	}
	fake := ssh(t, []string{written(t, &promov1.ListDueTeasersResponse{Teasers: []*promov1.Teaser{finale}})}, "", 0)

	due, err := backend(t, fake, nil).Due(t.Context())

	require.NoError(t, err)
	require.Len(t, due, 1)
	assert.Equal(t, teaser_job.BattleID("season-1/finale"), due[0].Battle)
	assert.Equal(t, start, due[0].StartsAt)
	assert.Equal(t, 2, due[0].Cards)
	var kept promov1.Teaser
	require.NoError(t, protojson.Unmarshal(due[0].Teaser, &kept))
	assert.True(t, proto.Equal(finale, &kept))
	assert.Equal(t, "promo.v1.AdminService/ListDueTeasers", fake.call(t, 1, &promov1.ListDueTeasersRequest{}))
}

func TestARefusalSaysWhyAndAnSSHThatFailsIsBusy(t *testing.T) {
	_, err := backend(t, ssh(t, nil, "wget: server returned error: HTTP/1.1 400 Bad Request", 1), nil).Due(t.Context())
	require.ErrorContains(t, err, "the backend refused promo.v1.AdminService/ListDueTeasers")
	assert.NotErrorAs(t, err, new(*teaser_job.BusyError))

	_, err = backend(t, ssh(t, nil, "ssh: connect to host: Connection refused", 255), nil).Due(t.Context())
	var busy *teaser_job.BusyError
	require.ErrorAs(t, err, &busy)
	assert.Contains(t, busy.Reason, "Connection refused")

	_, err = backend(t, ssh(t, nil, "wget: server returned error: HTTP/1.1 503 Service Unavailable", 1), nil).Due(t.Context())
	require.ErrorAs(t, err, &busy, "a server that failed may be restarting")
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

func link(t *testing.T, server *httptest.Server, name string) string {
	t.Helper()
	return written(t, &publishingv1.CreateMediaUploadResponse{UploadUrl: server.URL + "/upload/" + name, MediaUrl: "https://media.example/" + name})
}

// A file of the renderer's, holding its own name.
func file(t *testing.T, name string) teaser_job.File {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(name), 0o600))
	return teaser_job.File(path)
}

func TestAFileIsUploadedOnceThroughTheLinkPublishingSigns(t *testing.T) {
	kept, server := serve(t, http.StatusOK)
	fake := ssh(t, []string{link(t, server, "card-1.jpg")}, "", 0)
	uploads := backend(t, fake, server.Client())
	card := file(t, "card-1.jpg")

	for range 2 {
		address, err := uploads.Upload(t.Context(), card, "image/jpeg")
		require.NoError(t, err)
		assert.Equal(t, teaser_job.MediaURL("https://media.example/card-1.jpg"), address)
	}

	assert.Equal(t, 1, fake.calls(t), "the second is the first's")
	var asked publishingv1.CreateMediaUploadRequest
	assert.Equal(t, "publishing.v1.AdminService/CreateMediaUpload", fake.call(t, 1, &asked))
	assert.True(t, proto.Equal(&publishingv1.CreateMediaUploadRequest{FileName: "card-1.jpg", ContentType: "image/jpeg", SizeBytes: 10}, &asked))
	assert.Equal(t, map[string]string{"/upload/card-1.jpg": "image/jpeg card-1.jpg"}, kept.puts)
}

func TestAnUploadTheStorageRefusesSaysSoWithoutItsAddressAndABusyOneIsBusy(t *testing.T) {
	_, refusing := serve(t, http.StatusForbidden)
	_, err := backend(t, ssh(t, []string{link(t, refusing, "teaser.mp4")}, "", 0), refusing.Client()).Upload(t.Context(), file(t, "teaser.mp4"), "video/mp4")
	require.EqualError(t, err, "the storage answered 403 to the upload of a file")

	_, busy := serve(t, http.StatusServiceUnavailable)
	_, err = backend(t, ssh(t, []string{link(t, busy, "teaser.mp4")}, "", 0), busy.Client()).Upload(t.Context(), file(t, "teaser.mp4"), "video/mp4")
	require.ErrorAs(t, err, new(*teaser_job.BusyError))
}

func TestATeaserIsHandedToPromoWithItsMediaInOrder(t *testing.T) {
	fake := ssh(t, []string{written(t, &promov1.PostTeaserResponse{Outcome: &promov1.PostTeaserResponse_Posted{Posted: &promov1.TeaserPosted{
		PostId: "post-1", Title: "BATTLE", At: timestamppb.New(at), Platforms: []string{"tiktok", "youtube"},
		Refusals: []*promov1.TeaserRefusal{{Platform: "youtube", Reason: "no account"}},
	}}})}, "", 0)
	media := teaser_job.Media{Photos: []teaser_job.MediaURL{"https://media.example/card-1.jpg", "https://media.example/card-2.jpg"}, Video: "https://media.example/teaser.mp4"}

	outcome, err := backend(t, fake, nil).Post(t.Context(), "entry-7", media)

	require.NoError(t, err)
	assert.Equal(t, teaser_job.Outcome{
		Battle: "entry-7", Result: teaser_job.Posted, Title: "BATTLE", At: at, Platforms: []string{"tiktok", "youtube"}, Post: "post-1",
		Refused: map[string]string{"youtube": "no account"},
	}, outcome)
	var asked promov1.PostTeaserRequest
	assert.Equal(t, "promo.v1.AdminService/PostTeaser", fake.call(t, 1, &asked))
	assert.True(t, proto.Equal(&promov1.PostTeaserRequest{
		BattleId: "entry-7", PhotoUrls: []string{"https://media.example/card-1.jpg", "https://media.example/card-2.jpg"},
		VideoUrl: "https://media.example/teaser.mp4",
	}, &asked))
}

func TestWhatPromoAnswersIsTheOutcomeAndBusyIsAWait(t *testing.T) {
	for _, each := range []struct {
		said    *promov1.PostTeaserResponse
		outcome teaser_job.Outcome
	}{
		{
			said:    &promov1.PostTeaserResponse{Outcome: &promov1.PostTeaserResponse_Held{Held: &promov1.TeaserNotPosted{Reason: "no room"}}},
			outcome: teaser_job.Outcome{Battle: "entry-7", Result: teaser_job.Held, Why: "no room"},
		},
		{
			said:    &promov1.PostTeaserResponse{Outcome: &promov1.PostTeaserResponse_Skipped{Skipped: &promov1.TeaserNotPosted{Reason: "teased already"}}},
			outcome: teaser_job.Outcome{Battle: "entry-7", Result: teaser_job.Skipped, Why: "teased already"},
		},
		{
			said:    &promov1.PostTeaserResponse{Outcome: &promov1.PostTeaserResponse_Failed{Failed: &promov1.TeaserNotPosted{Reason: "refused"}}},
			outcome: teaser_job.Outcome{Battle: "entry-7", Result: teaser_job.Failed, Why: "refused"},
		},
	} {
		outcome, err := backend(t, ssh(t, []string{written(t, each.said)}, "", 0), nil).Post(t.Context(), "entry-7", teaser_job.Media{})
		require.NoError(t, err)
		assert.Equal(t, each.outcome, outcome)
	}

	busy := written(t, &promov1.PostTeaserResponse{Outcome: &promov1.PostTeaserResponse_Busy{Busy: &promov1.TeaserBusy{
		Reason: "zernio answered 429", RetryAfter: durationpb.New(time.Minute),
	}}})
	_, err := backend(t, ssh(t, []string{busy}, "", 0), nil).Post(t.Context(), "entry-7", teaser_job.Media{})
	var wait *teaser_job.BusyError
	require.ErrorAs(t, err, &wait)
	assert.Equal(t, &teaser_job.BusyError{RetryAfter: time.Minute, Reason: "zernio answered 429"}, wait)

	_, err = backend(t, ssh(t, []string{`{}`}, "", 0), nil).Post(t.Context(), "entry-7", teaser_job.Media{})
	require.EqualError(t, err, "promo said nothing of the teaser")
}

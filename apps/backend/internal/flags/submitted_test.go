package flags_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/postgres_flag_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed/usecases/tell_streams_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/postgres_submission_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/usecases/approve_submission_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/usecases/open_submitted_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/usecases/submit_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/subscribers/told_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpoutbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis/cpredistest"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// A player's flag goes from its submission, changed while it waits, to every player's open page with no restart.
func TestASubmittedFlagChangedWhileItWaitsOpensForEveryPlayer(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "flags", migrations.FS)
	streams := cpredis.New(cpredistest.StartServer(t).Config(), prometheus.NewRegistry())
	require.NoError(t, streams.ConnectCtx(t.Context()))
	t.Cleanup(func() { _ = streams.Close() })

	flags, submitted := postgres_flag_store.New(db), postgres_submission_store.New(db)
	clock := cptime.NewFixedClock(time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC))
	board := feed.NewBoard(8)
	page, leave := board.Attend()
	t.Cleanup(leave)
	tell := tell_streams_usecase.New(feed.NewsSince(clock.Now()), flags, board)
	opened, err := cpredis.Subscribe(t.Context(), streams, "flags-streams-opened", told_subscriber.NewOpened(tell))
	require.NoError(t, err)
	running(t, opened, cpoutbox.NewRelay("flags-outbox", db, streams, slog.New(slog.DiscardHandler)))

	bucket := pictures.NewMemoryBucket()
	gallery := pictures.NewGallery(bucket, pictures.Rules{MaxBytes: 5 << 20})
	submit := submit_flag_usecase.New(&submissions.Sequence{}, clock, gallery, db, submitted)
	approve := approve_submission_usecase.New(clock, gallery, db,
		approve_submission_usecase.Stores{Submissions: submitted, Flags: flags}, cpoutbox.New(db))
	founder := "00000000-0000-0000-0000-000000000001"

	bucket.Upload(pictures.UploadID{1}, pictures.TestPNG(300, 200))
	rennes, err := submit.Execute(t.Context(), submit_flag_usecase.In{
		Account: founder, Linked: true, Name: "Rennes", Upload: pictures.UploadID{1}.String(), Latitude: 48.1173,
		Longitude: -1.6778,
	})
	require.NoError(t, err)
	assert.Empty(t, bucket.Published(), "private until its review")

	clock.Advance(time.Hour)
	bucket.Upload(pictures.UploadID{2}, pictures.TestPNG(400, 200))
	breizh, err := submit.Execute(t.Context(), submit_flag_usecase.In{
		Account: founder, Linked: true, Name: "Breizh", Upload: pictures.UploadID{2}.String(), Latitude: 48.1173,
		Longitude: -1.6778, Replaces: rennes.ID.String(),
	})
	require.NoError(t, err)
	_, err = approve.Execute(t.Context(), approve_submission_usecase.In{Submission: rennes.ID.String()})
	require.ErrorIs(t, err, submissions.ErrUnknownSubmission, "the operator never approves what it replaced")

	_, err = approve.Execute(t.Context(), approve_submission_usecase.In{Submission: breizh.ID.String()})
	require.NoError(t, err)
	flag, err := open_submitted_flag_usecase.New(catalog.NewOpening(&catalog.Sequence{}, clock), db, submitted, flags,
		cpoutbox.New(db)).Execute(t.Context(), open_submitted_flag_usecase.In{
		Submission: breizh.ID.String(), EndsAt: clock.Now().Add(30 * 24 * time.Hour),
	})
	require.NoError(t, err)
	assert.Equal(t, catalog.Pin{Latitude: 48.1173, Longitude: -1.6778}, flag.Pin)
	told := <-page
	assert.Equal(t, catalog.Name("Breizh"), told.Name, "the page hears it opened")
	assert.Equal(t, [2]uint32{400, 200}, [2]uint32{told.Picture.Width, told.Picture.Height})

	bucket.Upload(pictures.UploadID{3}, pictures.TestPNG(400, 200))
	_, err = submit.Execute(t.Context(), submit_flag_usecase.In{
		Account: founder, Linked: true, Name: "Bretagne", Upload: pictures.UploadID{3}.String(), Latitude: 48.1173,
		Longitude: -1.6778, Replaces: breizh.ID.String(),
	})
	require.ErrorIs(t, err, submissions.ErrNotWaiting, "an opened flag never changes")
}

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

// A player's flag goes from its submission to every player, and its founder's new name reaches the open pages with no
// restart, its old one kept until the change is approved.
func TestASubmittedFlagOpensAndItsApprovedChangeReachesTheOpenPages(t *testing.T) {
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
	renamed, err := cpredis.Subscribe(t.Context(), streams, "flags-streams-renamed", told_subscriber.NewRenamed(tell))
	require.NoError(t, err)
	running(t, opened, renamed, cpoutbox.NewRelay("flags-outbox", db, streams, slog.New(slog.DiscardHandler)))

	bucket := pictures.NewMemoryBucket()
	gallery := pictures.NewGallery(bucket, pictures.Rules{MaxBytes: 5 << 20})
	submit := submit_flag_usecase.New(&submissions.Sequence{}, clock, gallery, submitted)
	approve := approve_submission_usecase.New(clock, gallery, db, submitted, flags, cpoutbox.New(db))
	founder := "00000000-0000-0000-0000-000000000001"

	bucket.Upload(pictures.UploadID{1}, pictures.TestPNG(300, 200))
	rennes, err := submit.Execute(t.Context(), submit_flag_usecase.In{
		Account: founder, Linked: true, Name: "Rennes", Upload: pictures.UploadID{1}.String(), Latitude: 48.1173,
		Longitude: -1.6778,
	})
	require.NoError(t, err)
	assert.Empty(t, bucket.Published(), "private until its review")
	_, err = approve.Execute(t.Context(), approve_submission_usecase.In{Submission: rennes.ID.String()})
	require.NoError(t, err)
	flag, err := open_submitted_flag_usecase.New(catalog.NewOpening(&catalog.Sequence{}, clock), db, submitted, flags,
		cpoutbox.New(db)).Execute(t.Context(), open_submitted_flag_usecase.In{
		Submission: rennes.ID.String(), EndsAt: clock.Now().Add(30 * 24 * time.Hour),
	})
	require.NoError(t, err)
	assert.Equal(t, catalog.Pin{Latitude: 48.1173, Longitude: -1.6778}, flag.Pin)
	assert.Equal(t, catalog.Name("Rennes"), (<-page).Name, "the page hears it opened")

	clock.Advance(time.Hour)
	bucket.Upload(pictures.UploadID{2}, pictures.TestPNG(400, 200))
	change, err := submit.Execute(t.Context(), submit_flag_usecase.In{
		Account: founder, Linked: true, Name: "Breizh", Upload: pictures.UploadID{2}.String(), Latitude: 48.1173,
		Longitude: -1.6778, Flag: flag.ID.String(),
	})
	require.NoError(t, err)
	kept, err := flags.Flag(t.Context(), flag.ID)
	require.NoError(t, err)
	assert.Equal(t, catalog.Name("Rennes"), kept.TakeSnapshot().Name, "the old name stays until the change's approval")

	_, err = approve.Execute(t.Context(), approve_submission_usecase.In{Submission: change.ID.String()})
	require.NoError(t, err)
	restyled := <-page
	assert.Equal(t, catalog.Name("Breizh"), restyled.Name, "the page hears the new name")
	assert.Equal(t, [2]uint32{400, 200}, [2]uint32{restyled.Picture.Width, restyled.Picture.Height})
}

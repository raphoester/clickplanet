package submit_flag_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/usecases/submit_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now    = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	upload = pictures.UploadID{1}
	ada    = submissions.AccountID{15: 1}
)

type fixture struct {
	store   *submissions.MemoryStore
	bucket  *pictures.MemoryBucket
	useCase *submit_flag_usecase.UseCase
}

func newFixture() fixture {
	store, bucket := submissions.NewMemoryStore(), pictures.NewMemoryBucket()
	bucket.Upload(upload, pictures.TestPNG(300, 200))
	return fixture{store: store, bucket: bucket, useCase: submit_flag_usecase.New(&submissions.Sequence{},
		cptime.NewFixedClock(now), pictures.NewGallery(bucket, pictures.Rules{MaxBytes: 5 << 20}), store)}
}

func rennes() submit_flag_usecase.In {
	return submit_flag_usecase.In{
		Account: ada.String(), Linked: true, Name: " Rennes ", Upload: upload.String(), Latitude: 48.1173,
		Longitude: -1.6778,
	}
}

func TestASubmissionIsKeptWaitingWithItsPictureHeldInPrivateAndItsUploadGoes(t *testing.T) {
	f := newFixture()

	submitted, err := f.useCase.Execute(t.Context(), rennes())
	require.NoError(t, err)

	id := submissions.SubmissionID{15: 1}
	assert.Equal(t, id, submitted.ID)
	assert.Equal(t, catalog.Name("Rennes"), submitted.Name)
	assert.Equal(t, catalog.Pin{Latitude: 48.1173, Longitude: -1.6778}, submitted.Pin)
	assert.Equal(t, submissions.Waiting, submitted.State)
	assert.Equal(t, [2]uint32{300, 200}, [2]uint32{submitted.Picture.Width, submitted.Picture.Height})
	assert.Empty(t, f.bucket.Published(), "nothing public before its review")
	assert.Contains(t, f.bucket.HeldPictures(), id.PictureKey())
	assert.Zero(t, f.bucket.Uploads())
	assert.Equal(t, []submissions.Snapshot{submitted}, f.store.Kept())
}

func TestAGuestSubmitsNothingAndItsUploadIsNotRead(t *testing.T) {
	f := newFixture()
	in := rennes()
	in.Linked = false

	_, err := f.useCase.Execute(t.Context(), in)

	require.ErrorIs(t, err, submissions.ErrGuest)
	assert.Equal(t, 1, f.bucket.Uploads())
	assert.Empty(t, f.store.Kept())
}

func TestASecondSubmissionWaitsForTheFirstsReview(t *testing.T) {
	f := newFixture()
	f.store.Seed(submissions.TestWaiting(submissions.SubmissionID{15: 9}, ada, now.Add(-time.Hour)))

	_, err := f.useCase.Execute(t.Context(), rennes())

	require.ErrorIs(t, err, submissions.ErrWaiting)
	assert.Empty(t, f.bucket.HeldPictures())
	assert.Len(t, f.store.Kept(), 1)
}

func TestAPictureNoFlagCanWearIsRefused(t *testing.T) {
	f := newFixture()
	f.bucket.Upload(upload, pictures.TestPNG(500, 200))

	_, err := f.useCase.Execute(t.Context(), rennes())

	require.ErrorIs(t, err, pictures.ErrShape)
	assert.Empty(t, f.store.Kept())
	assert.Equal(t, 1, f.bucket.Uploads(), "the player may send another")
}

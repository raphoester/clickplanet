package discard_unsent_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/inmemory_attachment_bucket"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/uploads"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/uploads/inmemory_upload_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/uploads/usecases/discard_unsent_usecase"
)

var noon = time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

const day = 24 * time.Hour

func file(n int) notes.Attachment {
	return notes.Attachment{ID: notes.AttachmentID{14: byte(n >> 8), 15: byte(n)}, MediaType: notes.MediaPNG, Size: 10}
}

// Granted at, put in the bucket.
func granted(t *testing.T, storage *inmemory_upload_storage.Storage, bucket *inmemory_attachment_bucket.Bucket, at time.Time, files ...notes.Attachment) {
	t.Helper()

	shelf := uploads.NewShelf(storage, 1<<40, cptime.NewFixedClock(at))
	for _, sent := range files {
		require.NoError(t, shelf.Reserve(t.Context(), sent))
	}
	bucket.Put(files...)
}

func TestAFileNoNoteNamedForADayIsDeletedAndItsRoomFreed(t *testing.T) {
	storage, bucket := inmemory_upload_storage.New(), inmemory_attachment_bucket.New()
	granted(t, storage, bucket, noon.Add(-25*time.Hour), file(1), file(2))
	granted(t, storage, bucket, noon.Add(-time.Hour), file(3))
	storage.Send(file(2).ID)

	discarded, err := discard_unsent_usecase.New(day, cptime.NewFixedClock(noon), storage, bucket).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, int64(1), discarded)
	_, err = bucket.Attachments(t.Context(), file(1).ID)
	require.ErrorIs(t, err, notes.ErrNotUploaded, "the file a day old that no note names")
	held, err := bucket.Attachments(t.Context(), file(2).ID, file(3).ID)
	require.NoError(t, err, "a file a note names, and a file that may still be named")
	assert.Len(t, held, 2)
	assert.Equal(t, []notes.AttachmentID{file(2).ID, file(3).ID}, idsOf(storage.Kept()))
}

func TestEveryBatchIsDeletedInOneRun(t *testing.T) {
	storage, bucket := inmemory_upload_storage.New(), inmemory_attachment_bucket.New()
	files := make([]notes.Attachment, 0, 250)
	for n := range 250 {
		files = append(files, file(n+1))
	}
	granted(t, storage, bucket, noon.Add(-2*day), files...)

	discarded, err := discard_unsent_usecase.New(day, cptime.NewFixedClock(noon), storage, bucket).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, int64(250), discarded)
	assert.Empty(t, storage.Kept())
}

func TestAFileTheBucketWouldNotDeleteKeepsItsRoomForTheNextRun(t *testing.T) {
	storage, bucket := inmemory_upload_storage.New(), inmemory_attachment_bucket.New()
	granted(t, storage, bucket, noon.Add(-2*day), file(1))
	down := errors.New("the bucket is down")
	bucket.FailWith(down)

	discarded, err := discard_unsent_usecase.New(day, cptime.NewFixedClock(noon), storage, bucket).Execute(t.Context())

	require.ErrorIs(t, err, down)
	assert.Zero(t, discarded)
	assert.Len(t, storage.Kept(), 1)
}

func TestNothingToDiscardAsksNothingOfTheBucket(t *testing.T) {
	bucket := inmemory_attachment_bucket.New()
	bucket.FailWith(errors.New("no bucket"))

	discarded, err := discard_unsent_usecase.New(day, cptime.NewFixedClock(noon), inmemory_upload_storage.New(), bucket).
		Execute(t.Context())

	require.NoError(t, err)
	assert.Zero(t, discarded)
}

func idsOf(kept []uploads.Snapshot) []notes.AttachmentID {
	ids := make([]notes.AttachmentID, 0, len(kept))
	for _, snapshot := range kept {
		ids = append(ids, snapshot.ID)
	}
	return ids
}

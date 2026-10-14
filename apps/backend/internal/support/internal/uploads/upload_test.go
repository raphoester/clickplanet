package uploads_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/uploads"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/uploads/inmemory_upload_storage"
)

var noon = time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

func file(n byte, size int64) notes.Attachment {
	return notes.Attachment{ID: notes.AttachmentID{15: n}, MediaType: notes.MediaMP4, Size: size}
}

func TestAnUploadIsAFileItsSizeAndWhenItsLinkWasSigned(t *testing.T) {
	upload, err := uploads.NewUpload(file(1, 400), noon)

	require.NoError(t, err)
	assert.Equal(t, uploads.Snapshot{ID: notes.AttachmentID{15: 1}, Size: 400, At: noon}, upload.TakeSnapshot())
}

func TestASnapshotThatBreaksARuleRestoresNothing(t *testing.T) {
	for name, broken := range map[string]uploads.Snapshot{
		"no id":   {Size: 400, At: noon},
		"no size": {ID: notes.AttachmentID{15: 1}, At: noon},
		"no time": {ID: notes.AttachmentID{15: 1}, Size: 400},
	} {
		_, err := broken.Restore()

		require.ErrorIs(t, err, uploads.ErrInvalidUpload, name)
	}
}

func TestADiscardedUploadIsACopyMarkedSo(t *testing.T) {
	upload, err := uploads.NewUpload(file(1, 400), noon)
	require.NoError(t, err)

	discarded := uploads.AllDiscarded([]uploads.Upload{upload})

	require.Len(t, discarded, 1)
	assert.True(t, discarded[0].TakeSnapshot().Discarded)
	assert.False(t, upload.TakeSnapshot().Discarded, "the upload it was made from is left as it was")
	assert.Equal(t, []notes.AttachmentID{{15: 1}}, uploads.FilesOf(discarded))
}

func TestTheShelfKeepsRoomForAFileUntilItIsFull(t *testing.T) {
	storage := inmemory_upload_storage.New()
	shelf := uploads.NewShelf(storage, 1000, cptime.NewFixedClock(noon))

	require.NoError(t, shelf.Reserve(t.Context(), file(1, 600)))
	room, err := shelf.Room(t.Context())
	require.NoError(t, err)
	assert.Equal(t, int64(400), room)

	require.ErrorIs(t, shelf.Reserve(t.Context(), file(2, 401)), uploads.ErrShelfFull)
	require.NoError(t, shelf.Reserve(t.Context(), file(3, 400)))

	room, err = shelf.Room(t.Context())
	require.NoError(t, err)
	assert.Equal(t, int64(0), room)
	assert.Equal(t, []uploads.Snapshot{
		{ID: notes.AttachmentID{15: 1}, Size: 600, At: noon},
		{ID: notes.AttachmentID{15: 3}, Size: 400, At: noon},
	}, storage.Kept())
}

func TestAShelfWithNoRoomLeftSaysNoneRatherThanLess(t *testing.T) {
	storage := inmemory_upload_storage.New()
	require.NoError(t, uploads.NewShelf(storage, 1000, cptime.NewFixedClock(noon)).Reserve(t.Context(), file(1, 900)))

	room, err := uploads.NewShelf(storage, 500, cptime.NewFixedClock(noon)).Room(t.Context())

	require.NoError(t, err)
	assert.Equal(t, int64(0), room, "a smaller shelf than what is held")
}

func TestTheShelfPassesOnAStoreThatFails(t *testing.T) {
	storage := inmemory_upload_storage.New()
	broken := errors.New("the database is gone")
	storage.FailWith(broken)
	shelf := uploads.NewShelf(storage, 1000, cptime.NewFixedClock(noon))

	require.ErrorIs(t, shelf.Reserve(t.Context(), file(1, 1)), broken)
	_, err := shelf.Room(t.Context())
	require.ErrorIs(t, err, broken)
}

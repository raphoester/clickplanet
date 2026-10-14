package create_upload_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/inmemory_attachment_bucket"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/create_upload_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/uploads"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/uploads/inmemory_upload_storage"
)

var (
	rules = notes.AttachmentRules{MaxPerNote: 10, MaxBytes: 1000}
	noon  = time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
)

type failingIDs struct{}

func (failingIDs) NewAttachmentID() (notes.AttachmentID, error) {
	return notes.AttachmentID{}, errors.New("no entropy")
}

func shelfOf(storage *inmemory_upload_storage.Storage, capacity int64) *uploads.Shelf {
	return uploads.NewShelf(storage, capacity, cptime.NewFixedClock(noon))
}

func TestAFileTheRulesTakeGetsAFreshIDARoomAndALinkToUploadItTo(t *testing.T) {
	ids := &notes.SequentialIDs{}
	storage := inmemory_upload_storage.New()
	useCase := create_upload_usecase.New(ids, rules, shelfOf(storage, 10_000), inmemory_attachment_bucket.New())

	first, err := useCase.Execute(t.Context(), create_upload_usecase.In{MediaType: notes.MediaPNG, Size: 400})
	require.NoError(t, err)
	second, err := useCase.Execute(t.Context(), create_upload_usecase.In{MediaType: notes.MediaMP4, Size: 1000})
	require.NoError(t, err)

	assert.Equal(t, create_upload_usecase.Out{
		ID: notes.AttachmentID{0: 0xa, 15: 1}, URL: inmemory_attachment_bucket.URL(notes.AttachmentID{0: 0xa, 15: 1}),
	}, first)
	assert.NotEqual(t, first.ID, second.ID)
	assert.Equal(t, []uploads.Snapshot{
		{ID: first.ID, Size: 400, At: noon},
		{ID: second.ID, Size: 1000, At: noon},
	}, storage.Kept())
}

func TestAFileTheRulesDoNotTakeGetsNoLinkAndNoRoom(t *testing.T) {
	storage := inmemory_upload_storage.New()
	for name, in := range map[string]create_upload_usecase.In{
		"too big":              {MediaType: notes.MediaMP4, Size: 1001},
		"not a photo or video": {MediaType: "text/html", Size: 10},
		"empty":                {MediaType: notes.MediaPNG},
	} {
		_, err := create_upload_usecase.New(&notes.SequentialIDs{}, rules, shelfOf(storage, 10_000), inmemory_attachment_bucket.New()).
			Execute(t.Context(), in)

		require.ErrorIs(t, err, notes.ErrInvalidAttachment, name)
	}

	_, err := create_upload_usecase.New(&notes.SequentialIDs{}, notes.AttachmentRules{}, shelfOf(storage, 10_000),
		inmemory_attachment_bucket.New()).Execute(t.Context(), create_upload_usecase.In{MediaType: notes.MediaPNG, Size: 10})
	require.ErrorIs(t, err, notes.ErrInvalidAttachment, "a server that takes no file")
	assert.Empty(t, storage.Kept())
}

func TestAFileTheBucketHasNoRoomForGetsNoLink(t *testing.T) {
	storage := inmemory_upload_storage.New()
	useCase := create_upload_usecase.New(&notes.SequentialIDs{}, rules, shelfOf(storage, 1500), inmemory_attachment_bucket.New())

	_, err := useCase.Execute(t.Context(), create_upload_usecase.In{MediaType: notes.MediaMP4, Size: 1000})
	require.NoError(t, err)
	_, err = useCase.Execute(t.Context(), create_upload_usecase.In{MediaType: notes.MediaMP4, Size: 1000})

	require.ErrorIs(t, err, uploads.ErrShelfFull)
	assert.Len(t, storage.Kept(), 1)
}

func TestAnUploadWithNoIDNoRoomReadOrNoLinkFails(t *testing.T) {
	_, err := create_upload_usecase.New(failingIDs{}, rules, shelfOf(inmemory_upload_storage.New(), 10_000),
		inmemory_attachment_bucket.New()).Execute(t.Context(), create_upload_usecase.In{MediaType: notes.MediaPNG, Size: 10})
	require.Error(t, err)

	gone := errors.New("the database is gone")
	storage := inmemory_upload_storage.New()
	storage.FailWith(gone)
	_, err = create_upload_usecase.New(&notes.SequentialIDs{}, rules, shelfOf(storage, 10_000), inmemory_attachment_bucket.New()).
		Execute(t.Context(), create_upload_usecase.In{MediaType: notes.MediaPNG, Size: 10})
	require.ErrorIs(t, err, gone)

	down := errors.New("no bucket")
	bucket := inmemory_attachment_bucket.New()
	bucket.FailWith(down)
	_, err = create_upload_usecase.New(&notes.SequentialIDs{}, rules, shelfOf(inmemory_upload_storage.New(), 10_000), bucket).
		Execute(t.Context(), create_upload_usecase.In{MediaType: notes.MediaPNG, Size: 10})
	require.ErrorIs(t, err, down)
}

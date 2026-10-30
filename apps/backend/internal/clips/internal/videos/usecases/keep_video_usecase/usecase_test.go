package keep_video_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos/usecases/create_upload_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos/usecases/keep_video_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos/usecases/remove_video_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	rules = videos.Rules{MaxBytes: 1 << 20}
	noon  = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
)

type world struct {
	bucket *videos.MemoryBucket
	store  *videos.MemoryStore
	create *create_upload_usecase.UseCase
	keep   *keep_video_usecase.UseCase
	remove *remove_video_usecase.UseCase
}

func newWorld() world {
	bucket, store := videos.NewMemoryBucket(), videos.NewMemoryStore()
	return world{
		bucket: bucket,
		store:  store,
		create: create_upload_usecase.New(&videos.SequentialIDs{}, rules, bucket),
		keep:   keep_video_usecase.New(bucket, store, rules, cptime.NewFixedClock(noon)),
		remove: remove_video_usecase.New(bucket, store),
	}
}

func TestAVideoSentThroughItsLinkIsKeptWithWhatTheBucketHolds(t *testing.T) {
	w := newWorld()
	upload, err := w.create.Execute(create_upload_usecase.In{MediaType: "video/mp4", Size: 5000})
	require.NoError(t, err)
	assert.Contains(t, upload.URL, upload.ID.String())

	held := videos.File{ID: upload.ID, MediaType: "video/mp4", Size: 4999}
	w.bucket.Put(held)
	require.NoError(t, w.keep.Execute(t.Context(),
		keep_video_usecase.In{ID: upload.ID, Name: "Season 0", Caption: "23 days."}))

	kept, ok := w.store.Kept(upload.ID)
	require.True(t, ok)
	assert.Equal(t, videos.VideoSnapshot{File: held, Name: "Season 0", Caption: "23 days.", KeptAt: noon}, kept,
		"the size is the bucket's, not the page's")
}

func TestNoLinkIsMadeForAFileThatIsNotAVideo(t *testing.T) {
	_, err := newWorld().create.Execute(create_upload_usecase.In{MediaType: "image/png", Size: 5000})
	require.ErrorIs(t, err, videos.ErrInvalidVideo)
}

func TestAVideoNeverSentIsNotKept(t *testing.T) {
	w := newWorld()
	upload, err := w.create.Execute(create_upload_usecase.In{MediaType: "video/mp4", Size: 5000})
	require.NoError(t, err)

	err = w.keep.Execute(t.Context(), keep_video_usecase.In{ID: upload.ID, Name: "Season 0"})

	require.ErrorIs(t, err, videos.ErrNotUploaded)
	_, ok := w.store.Kept(upload.ID)
	assert.False(t, ok)
}

func TestARemovedVideoLosesItsFileAndItsRow(t *testing.T) {
	w := newWorld()
	held := videos.File{ID: videos.VideoID{9}, MediaType: "video/webm", Size: 10}
	w.bucket.Put(held)
	require.NoError(t, w.keep.Execute(t.Context(), keep_video_usecase.In{ID: held.ID, Name: "Battle"}))

	require.NoError(t, w.remove.Execute(t.Context(), held.ID))

	_, ok := w.store.Kept(held.ID)
	assert.False(t, ok)
	assert.Equal(t, []videos.VideoID{held.ID}, w.bucket.Deleted)
	require.ErrorIs(t, w.remove.Execute(t.Context(), held.ID), videos.ErrNoVideo)
}

func TestAVideoWhoseFileCouldNotBeDeletedStaysListed(t *testing.T) {
	w := newWorld()
	held := videos.File{ID: videos.VideoID{9}, MediaType: "video/webm", Size: 10}
	w.bucket.Put(held)
	require.NoError(t, w.keep.Execute(t.Context(), keep_video_usecase.In{ID: held.ID, Name: "Battle"}))

	w.bucket.Failing(videos.ErrBucketDown)
	require.ErrorIs(t, w.remove.Execute(t.Context(), held.ID), videos.ErrBucketDown)

	_, ok := w.store.Kept(held.ID)
	assert.True(t, ok, "asked again, it is removed")
}

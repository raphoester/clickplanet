package pictures_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
)

const key = "flags/submissions/00000000-0000-0000-0000-000000000001.png"

func TestAPictureHeldIsPublicOnlyOnceReleased(t *testing.T) {
	bucket := pictures.NewMemoryBucket()
	gallery := pictures.NewGallery(bucket, pictures.Rules{MaxBytes: 5 << 20})
	bucket.Upload(pictures.UploadID{1}, pictures.TestPNG(300, 200))

	drawn, err := gallery.Redrawn(t.Context(), pictures.UploadID{1})
	require.NoError(t, err)
	require.NoError(t, gallery.Hold(t.Context(), key, drawn))
	assert.Empty(t, bucket.Published())
	assert.Equal(t, drawn.PNG, bucket.HeldPictures()[key])

	require.NoError(t, gallery.Release(t.Context(), key, drawn.Picture))
	assert.Equal(t, drawn.PNG, bucket.Published()[drawn.Picture.Key()])
}

func TestAPictureHeldThatIsNotTheOneApprovedIsNotReleased(t *testing.T) {
	bucket := pictures.NewMemoryBucket()
	gallery := pictures.NewGallery(bucket, pictures.Rules{MaxBytes: 5 << 20})
	drawn, err := pictures.Redraw(pictures.TestPNG(300, 200))
	require.NoError(t, err)
	other, err := pictures.Redraw(pictures.TestPNG(400, 200))
	require.NoError(t, err)
	require.NoError(t, gallery.Hold(t.Context(), key, other))

	require.ErrorIs(t, gallery.Release(t.Context(), key, drawn.Picture), pictures.ErrNotThePicture)
	require.ErrorIs(t, gallery.Release(t.Context(), "flags/submissions/none.png", drawn.Picture), pictures.ErrUnknownHeld)
	assert.Empty(t, bucket.Published())
}

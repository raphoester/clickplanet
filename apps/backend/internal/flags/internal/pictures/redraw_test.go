package pictures_test

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image/png"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
)

func drawnSize(t *testing.T, drawn pictures.Drawn) (int, int) {
	t.Helper()
	decoded, err := png.Decode(bytes.NewReader(drawn.PNG))
	require.NoError(t, err, "the server writes a PNG whatever it was sent")
	return decoded.Bounds().Dx(), decoded.Bounds().Dy()
}

func TestAPNGAJPEGAndAWebPAreDrawnAgainAsAPNG(t *testing.T) {
	webp, err := os.ReadFile("testdata/bands.webp")
	require.NoError(t, err)

	for name, file := range map[string][]byte{
		"png": pictures.TestPNG(300, 200), "jpeg": pictures.TestJPEG(300, 200), "webp": webp,
	} {
		drawn, err := pictures.Redraw(file)
		require.NoError(t, err, name)
		width, height := drawnSize(t, drawn)
		assert.Equal(t, [2]int{300, 200}, [2]int{width, height}, name)
		kept := drawn.Picture.TakeSnapshot()
		assert.Equal(t, [2]uint32{300, 200}, [2]uint32{kept.Width, kept.Height}, name)
		assert.Regexp(t, `^flags/pictures/[0-9a-f]{64}\.png$`, drawn.Picture.Key(), name)
	}
}

func TestABigPictureIsFittedInside768By512(t *testing.T) {
	for _, size := range [][2]int{{1600, 1000}, {2000, 1000}, {900, 900}} {
		drawn, err := pictures.Redraw(pictures.TestPNG(size[0], size[1]))
		require.NoError(t, err, size)
		width, height := drawnSize(t, drawn)
		assert.LessOrEqual(t, width, 768, size)
		assert.LessOrEqual(t, height, 512, size)
		assert.InDelta(t, float64(size[0])/float64(size[1]), float64(width)/float64(height), 0.01, "its shape kept")
	}
}

func TestAPictureNarrowerThanSquareOrWiderThanTwoToOneIsRefused(t *testing.T) {
	for _, size := range [][2]int{{200, 201}, {401, 200}} {
		_, err := pictures.Redraw(pictures.TestPNG(size[0], size[1]))
		require.ErrorIs(t, err, pictures.ErrShape, size)
	}
	_, err := pictures.Redraw(pictures.TestPNG(99, 60))
	require.ErrorIs(t, err, pictures.ErrTooSmall)
}

func TestWhatIsNoPNGJPEGOrWebPIsRefused(t *testing.T) {
	for name, file := range map[string][]byte{
		"svg":   []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="300" height="200"><script>alert(1)</script></svg>`),
		"gif":   []byte("GIF89a\x01\x00\x01\x00"),
		"empty": {},
		"half":  pictures.TestPNG(300, 200)[:100],
	} {
		_, err := pictures.Redraw(file)
		require.ErrorIs(t, err, pictures.ErrNotAPicture, name)
	}
}

func TestAPictureThatMovesIsRefused(t *testing.T) {
	still := pictures.TestPNG(300, 200)
	animation := chunk("acTL", []byte{0, 0, 0, 2, 0, 0, 0, 0})
	const ihdrEnd = 8 + 12 + 13
	moving := append(append(append([]byte{}, still[:ihdrEnd]...), animation...), still[ihdrEnd:]...)
	_, err := pictures.Redraw(moving)
	require.ErrorIs(t, err, pictures.ErrMoves, "an APNG")

	webp := []byte("RIFF\x00\x00\x00\x00WEBPVP8X\x0a\x00\x00\x00\x02\x00\x00\x00")
	_, err = pictures.Redraw(webp)
	require.ErrorIs(t, err, pictures.ErrMoves, "an animated WebP")
}

func TestThePictureKeepsNothingButItsPixelsAndIsNamedAfterThem(t *testing.T) {
	still := pictures.TestPNG(300, 200)
	const ihdrEnd = 8 + 12 + 13
	text := chunk("tEXt", []byte("Author\x00someone's name"))
	tagged := append(append(append([]byte{}, still[:ihdrEnd]...), text...), still[ihdrEnd:]...)

	first, err := pictures.Redraw(tagged)
	require.NoError(t, err)
	second, err := pictures.Redraw(still)
	require.NoError(t, err)

	assert.NotContains(t, string(first.PNG), "someone's name")
	assert.Equal(t, second.Picture.Key(), first.Picture.Key(), "the same pixels, the same name")
}

func TestAFlagOfBandsStretchesAndItsFocusIsWhereItDiffers(t *testing.T) {
	drawn, err := pictures.Redraw(pictures.TestPNG(300, 200))
	require.NoError(t, err)

	fit := drawn.Picture.TakeSnapshot().Fit
	assert.True(t, fit.Stretch, "two bands are only bands")
	assert.InDelta(t, 0.5, fit.FocusX, 0.01)
	assert.InDelta(t, 0.5, fit.FocusY, 0.05)
}

func chunk(kind string, data []byte) []byte {
	out := make([]byte, 0, 12+len(data))
	out = binary.BigEndian.AppendUint32(out, uint32(len(data))) //nolint:gosec // a test chunk.
	out = append(out, kind...)
	out = append(out, data...)
	return binary.BigEndian.AppendUint32(out, crc32.ChecksumIEEE(append([]byte(kind), data...)))
}

func TestAFlagHoldsFewColoursAndAPhotoMany(t *testing.T) {
	bands, err := pictures.Redraw(pictures.TestPNG(600, 400))
	require.NoError(t, err)
	assert.Equal(t, pictures.OtherColors(0), bands.OtherColors)

	photo, err := pictures.Redraw(pictures.TestNoisePNG(600, 400))
	require.NoError(t, err)
	assert.Greater(t, photo.OtherColors, pictures.OtherColors(50))
}

package pictures

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/webp"
)

var (
	ErrNotAPicture = errors.New("not a PNG, JPEG or WebP picture")
	ErrMoves       = errors.New("a picture that moves is no flag")
	ErrShape       = errors.New("a flag is between square and 2 wide by 1 high")
	ErrTooSmall    = errors.New("a flag's picture is at least 100 pixels wide")
)

// The size of the big flags the far view draws.
const (
	maxWidth  = 768
	maxHeight = 512
	minWidth  = 100
)

// Drawn is a picture drawn again: a PNG with nothing but its pixels, fitted inside 768 x 512.
type Drawn struct {
	PNG     []byte
	Picture Picture
}

// Redraw draws a file uploaded again, so what is made public is a picture the server wrote and nothing it was sent.
func Redraw(file []byte) (Drawn, error) {
	decoded, err := decode(file)
	if err != nil {
		return Drawn{}, err
	}

	bounds := decoded.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	switch {
	case width < height || width > 2*height:
		return Drawn{}, fmt.Errorf("%w: %d x %d", ErrShape, width, height)
	case width < minWidth:
		return Drawn{}, fmt.Errorf("%w: %d", ErrTooSmall, width)
	}

	fitted := resized(decoded, fittedSize(width, height))
	var out bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&out, fitted); err != nil {
		return Drawn{}, fmt.Errorf("failed to write the picture: %w", err)
	}

	sum := sha256.Sum256(out.Bytes())
	small := resized(fitted, image.Pt(minWidth, int(math.Round(float64(minWidth)*float64(fitted.Bounds().Dy())/
		float64(fitted.Bounds().Dx())))))
	return Drawn{PNG: out.Bytes(), Picture: Picture{
		hash:   hex.EncodeToString(sum[:]),
		width:  uint32(fitted.Bounds().Dx()), //nolint:gosec // at most 768.
		height: uint32(fitted.Bounds().Dy()), //nolint:gosec // at most 512.
		fit:    fitOf(small),
	}}, nil
}

func fittedSize(width, height int) image.Point {
	scale := math.Min(1, math.Min(float64(maxWidth)/float64(width), float64(maxHeight)/float64(height)))
	return image.Pt(max(1, int(math.Round(float64(width)*scale))), max(1, int(math.Round(float64(height)*scale))))
}

func resized(picture image.Image, size image.Point) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, size.X, size.Y))
	if size == picture.Bounds().Size() {
		draw.Draw(out, out.Bounds(), picture, picture.Bounds().Min, draw.Src)
		return out
	}
	xdraw.CatmullRom.Scale(out, out.Bounds(), picture, picture.Bounds(), xdraw.Src, nil)
	return out
}

func decode(file []byte) (image.Image, error) {
	switch {
	case bytes.HasPrefix(file, []byte("\x89PNG\r\n\x1a\n")):
		if animatedPNG(file) {
			return nil, ErrMoves
		}
		decoded, err := png.Decode(bytes.NewReader(file))
		return checked(decoded, err)
	case bytes.HasPrefix(file, []byte{0xff, 0xd8, 0xff}):
		decoded, err := jpeg.Decode(bytes.NewReader(file))
		return checked(decoded, err)
	case len(file) >= 12 && string(file[:4]) == "RIFF" && string(file[8:12]) == "WEBP":
		if animatedWebP(file) {
			return nil, ErrMoves
		}
		decoded, err := webp.Decode(bytes.NewReader(file))
		return checked(decoded, err)
	}
	return nil, ErrNotAPicture
}

func checked(decoded image.Image, err error) (image.Image, error) {
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNotAPicture, err)
	}
	return decoded, nil
}

// An APNG names its frames in an acTL chunk, before its first IDAT.
func animatedPNG(file []byte) bool {
	for at := 8; at+8 <= len(file); {
		length := int(binary.BigEndian.Uint32(file[at : at+4]))
		kind := string(file[at+4 : at+8])
		switch kind {
		case "acTL":
			return true
		case "IDAT":
			return false
		}
		at += 12 + length
	}
	return false
}

// An extended WebP says it moves in its VP8X header's flags.
func animatedWebP(file []byte) bool {
	const animation = 0x02
	return len(file) > 20 && string(file[12:16]) == "VP8X" && file[20]&animation != 0
}

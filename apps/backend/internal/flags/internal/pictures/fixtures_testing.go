//go:build testing

package pictures

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
)

// TestPNG is a flag of two bands, black over white, width by height.
func TestPNG(width, height int) []byte {
	var out bytes.Buffer
	if err := png.Encode(&out, bands(width, height)); err != nil {
		panic(err)
	}
	return out.Bytes()
}

func TestJPEG(width, height int) []byte {
	var out bytes.Buffer
	if err := jpeg.Encode(&out, bands(width, height), nil); err != nil {
		panic(err)
	}
	return out.Bytes()
}

func bands(width, height int) *image.NRGBA {
	picture := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			shade := color.NRGBA{A: 255}
			if y >= height/2 {
				shade = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			}
			picture.SetNRGBA(x, y, shade)
		}
	}
	return picture
}

// TestNoisePNG is a picture of every colour, as a photo holds.
func TestNoisePNG(width, height int) []byte {
	picture := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			picture.SetNRGBA(x, y, color.NRGBA{
				R: uint8(x * 7 % 256), G: uint8(y * 13 % 256), B: uint8((x + y) * 31 % 256), A: 255, //nolint:gosec // a test's noise.
			})
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, picture); err != nil {
		panic(err)
	}
	return out.Bytes()
}

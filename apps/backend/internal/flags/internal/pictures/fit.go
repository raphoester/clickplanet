package pictures

import (
	"image"
	"math"
)

// What the page's scripts/generateFlagFit.mjs works out for a flag the game ships with, on a picture of the same 100
// pixels' width.
const (
	tolerance = 14
	inset     = 2
)

type Fit struct {
	// The flag is only bands, so it may be stretched to a country's shape.
	Stretch bool
	// The point a crop keeps, each from 0 to 1.
	FocusX float64
	FocusY float64
}

func fitOf(picture *image.NRGBA) Fit {
	width, height := picture.Bounds().Dx(), picture.Bounds().Dy()
	if width <= inset*2 || height <= inset*2 {
		return Fit{FocusX: 0.5, FocusY: 0.5}
	}
	return Fit{
		Stretch: uniformLines(picture, false) || uniformLines(picture, true),
		FocusX:  rounded(focusOf(picture, true)),
		FocusY:  rounded(focusOf(picture, false)),
	}
}

func rgb(picture *image.NRGBA, x, y int) [3]float64 {
	at := picture.PixOffset(x, y)
	return [3]float64{float64(picture.Pix[at]), float64(picture.Pix[at+1]), float64(picture.Pix[at+2])}
}

func uniformLines(picture *image.NRGBA, vertical bool) bool {
	outer, inner := picture.Bounds().Dy(), picture.Bounds().Dx()
	if vertical {
		outer, inner = inner, outer
	}
	for a := inset; a < outer-inset; a++ {
		lo, hi := [3]float64{255, 255, 255}, [3]float64{0, 0, 0}
		for b := inset; b < inner-inset; b++ {
			x, y := b, a
			if vertical {
				x, y = a, b
			}
			pixel := rgb(picture, x, y)
			for c := range 3 {
				lo[c], hi[c] = math.Min(lo[c], pixel[c]), math.Max(hi[c], pixel[c])
			}
		}
		for c := range 3 {
			if hi[c]-lo[c] > tolerance {
				return false
			}
		}
	}
	return true
}

func focusOf(picture *image.NRGBA, vertical bool) float64 {
	outer, inner := picture.Bounds().Dy(), picture.Bounds().Dx()
	if vertical {
		outer, inner = inner, outer
	}
	at := func(a, b int) [3]float64 {
		if vertical {
			return rgb(picture, a, b)
		}
		return rgb(picture, b, a)
	}

	mean := make([]float64, inner*3)
	for a := range outer {
		for b := range inner {
			pixel := at(a, b)
			for c := range 3 {
				mean[b*3+c] += pixel[c] / float64(outer)
			}
		}
	}

	total, weighted := 0.0, 0.0
	for a := range outer {
		apart := 0.0
		for b := range inner {
			pixel := at(a, b)
			for c := range 3 {
				apart += math.Abs(pixel[c] - mean[b*3+c])
			}
		}
		apart /= float64(inner)
		total += apart
		weighted += apart * (float64(a) + 0.5)
	}
	if total < 1e-6 {
		return 0.5
	}
	return weighted / total / float64(outer)
}

func rounded(value float64) float64 {
	return math.Round(value*10000) / 10000
}

package pictures

import (
	"image"
	"slices"
)

// OtherColors is the share of a picture, in percent, in colours other than its 8 most common, each channel in 16
// steps. The game's flags hold 0 to 10, a photo 20 or more.
type OtherColors uint32

const (
	steps = 16
	// All the pixels a reader cannot see are one colour, whatever their channels hold.
	clear = steps * steps * steps
	main  = 8
)

func otherColorsOf(picture *image.NRGBA) OtherColors {
	counts := make([]int, clear+1)
	bounds := picture.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			at := picture.PixOffset(x, y)
			pixel := picture.Pix[at : at+4 : at+4]
			if pixel[3] < 128 {
				counts[clear]++
				continue
			}
			counts[(int(pixel[0])/steps*steps+int(pixel[1])/steps)*steps+int(pixel[2])/steps]++
		}
	}

	slices.SortFunc(counts, func(a, b int) int { return b - a })
	covered := 0
	for _, count := range counts[:main] {
		covered += count
	}
	total := bounds.Dx() * bounds.Dy()
	return OtherColors((total - covered) * 100 / total) //nolint:gosec // 0 to 100.
}

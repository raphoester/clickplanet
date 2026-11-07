package graffiti

import (
	"math"
	"net/url"
	"strconv"

	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
)

const (
	// The drawing fills this share of a desktop's half height, as the Discord pings frame an attack.
	fill        = 0.4
	widestZoom  = 1
	closestZoom = 20
)

// The page opened on the drawing's middle, at the zoom that frames it: the frontend reads ?lat=&lon=&zoom=, with the
// north pole on +y and longitude 0 on -x.
func LinkOf(page string, points []drawings.Point) string {
	var sum drawings.Point
	for _, point := range points {
		sum = drawings.Point{X: sum.X + point.X, Y: sum.Y + point.Y, Z: sum.Z + point.Z}
	}
	length := math.Sqrt(sum.X*sum.X + sum.Y*sum.Y + sum.Z*sum.Z)
	if length == 0 {
		return page
	}
	centre := drawings.Point{X: sum.X / length, Y: sum.Y / length, Z: sum.Z / length}

	framed := 0.0
	for _, point := range points {
		along := point.X*centre.X + point.Y*centre.Y + point.Z*centre.Z
		framed = max(framed, math.Sqrt(max(0, 1-along*along)))
	}
	zoom := float64(closestZoom)
	if framed > 0 {
		zoom = min(closestZoom, max(widestZoom, fill/framed))
	}

	return page + "?" + url.Values{
		"lat":  {strconv.FormatFloat(degrees(math.Asin(centre.Y)), 'f', 2, 64)},
		"lon":  {strconv.FormatFloat(degrees(math.Atan2(centre.Z, -centre.X)), 'f', 2, 64)},
		"zoom": {strconv.FormatFloat(zoom, 'f', 1, 64)},
	}.Encode()
}

func degrees(radians float64) float64 {
	return radians * 180 / math.Pi
}

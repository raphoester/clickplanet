package pings

import (
	"cmp"
	"math"
	"net/url"
	"slices"
	"strconv"
)

// On the unit globe, the frontend's: y is the north pole, and longitude 0 is on -x.
type Point struct{ X, Y, Z float64 }

// Where a link opens the globe, in degrees, at the camera's zoom: 1 at rest. The zero View is no place.
type View struct {
	Lat  float64
	Lon  float64
	Zoom float64
}

const (
	// A few stray takes neither move the view nor zoom it out.
	framedShare = 0.9
	// The framed takes fill this share of a desktop's half height, and most of a phone's half width.
	fill        = 0.4
	widestZoom  = 1
	closestZoom = 20
)

func viewOf(points []Point) View {
	centre, found := middleOf(points)
	if !found {
		return View{}
	}
	nearest := slices.Clone(points)
	slices.SortStableFunc(nearest, func(a, b Point) int { return cmp.Compare(offsetOf(a, centre), offsetOf(b, centre)) })
	nearest = nearest[:int(math.Ceil(framedShare*float64(len(nearest))))]
	centre, _ = middleOf(nearest)

	framed := 0.0
	for _, point := range nearest {
		framed = max(framed, offsetOf(point, centre))
	}
	zoom := float64(closestZoom)
	if framed > 0 {
		zoom = min(closestZoom, max(widestZoom, fill/framed))
	}
	return View{
		Lat:  degrees(math.Asin(centre.Y)),
		Lon:  degrees(math.Atan2(centre.Z, -centre.X)),
		Zoom: zoom,
	}
}

func middleOf(points []Point) (Point, bool) {
	var sum Point
	for _, point := range points {
		sum = Point{X: sum.X + point.X, Y: sum.Y + point.Y, Z: sum.Z + point.Z}
	}
	length := math.Sqrt(sum.X*sum.X + sum.Y*sum.Y + sum.Z*sum.Z)
	if length == 0 {
		return Point{}, false
	}
	return Point{X: sum.X / length, Y: sum.Y / length, Z: sum.Z / length}, true
}

// How far from the centre the point is drawn on a screen that looks at the centre.
func offsetOf(point, centre Point) float64 {
	along := point.X*centre.X + point.Y*centre.Y + point.Z*centre.Z
	return math.Sqrt(max(0, 1-along*along))
}

func (v View) linkFrom(page string) string {
	if v == (View{}) {
		return page
	}
	return page + "?" + url.Values{
		"lat":  {strconv.FormatFloat(v.Lat, 'f', 2, 64)},
		"lon":  {strconv.FormatFloat(v.Lon, 'f', 2, 64)},
		"zoom": {strconv.FormatFloat(v.Zoom, 'f', 1, 64)},
	}.Encode()
}

func degrees(radians float64) float64 {
	return radians * 180 / math.Pi
}

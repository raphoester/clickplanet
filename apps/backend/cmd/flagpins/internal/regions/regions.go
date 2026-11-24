// Package regions picks the capital of each flag of a region inside a country's ground, from Natural Earth.
package regions

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

var ErrNoCapital = errors.New("no place of Natural Earth inside the region")

// A flag of a region: its country's ISO code, and its ISO 3166-2 part (gb-eng, England).
var regionCode = regexp.MustCompile(`^([a-z]{2})-([a-z]{3})$`)

type Layer struct {
	Features []Feature `json:"features"`
}

type Feature struct {
	Properties map[string]any `json:"properties"`
	Geometry   Geometry       `json:"geometry"`
}

type Geometry struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

type Pin struct {
	Place     string  `json:"place"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Pins is the capital of each region among the codes: the place inside its map subunit that the map's capitals would
// pick, its main capital first, then a region's capital, then the biggest place.
func Pins(codes []string, subunits, places Layer) (map[string]Pin, error) {
	pins := map[string]Pin{}
	for _, code := range codes {
		parts := regionCode.FindStringSubmatch(code)
		if parts == nil {
			continue
		}
		country, part := strings.ToUpper(parts[1]), strings.ToUpper(parts[2])
		shape, err := subunitOf(subunits, country, part)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", code, err)
		}
		pin, err := capitalOf(places, country, shape)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", code, err)
		}
		pins[code] = pin
	}
	return pins, nil
}

// A polygon is rings of [longitude, latitude]: the first its edge, the others its holes.
type polygon [][][]float64

func subunitOf(subunits Layer, country, part string) ([]polygon, error) {
	for _, feature := range subunits.Features {
		if feature.Properties["ISO_A2_EH"] != country || feature.Properties["SU_A3"] != part {
			continue
		}
		switch feature.Geometry.Type {
		case "Polygon":
			var one polygon
			if err := json.Unmarshal(feature.Geometry.Coordinates, &one); err != nil {
				return nil, fmt.Errorf("failed to read the map subunit %s: %w", part, err)
			}
			return []polygon{one}, nil
		case "MultiPolygon":
			var many []polygon
			if err := json.Unmarshal(feature.Geometry.Coordinates, &many); err != nil {
				return nil, fmt.Errorf("failed to read the map subunit %s: %w", part, err)
			}
			return many, nil
		}
		return nil, fmt.Errorf("the map subunit %s is a %s", part, feature.Geometry.Type)
	}
	return nil, fmt.Errorf("no map subunit %s of %s", part, country)
}

type place struct {
	rank int
	pop  float64
	pin  Pin
}

func capitalOf(places Layer, country string, shape []polygon) (Pin, error) {
	var inside []place
	for _, feature := range places.Features {
		properties := feature.Properties
		if properties["ISO_A2"] != country {
			continue
		}
		latitude, _ := properties["LATITUDE"].(float64)
		longitude, _ := properties["LONGITUDE"].(float64)
		if !slices.ContainsFunc(shape, func(p polygon) bool { return p.holds(longitude, latitude) }) {
			continue
		}
		name, _ := properties["NAME"].(string)
		pop, _ := properties["POP_MAX"].(float64)
		inside = append(inside, place{
			rank: rankOf(properties), pop: pop, pin: Pin{Place: name, Latitude: latitude, Longitude: longitude},
		})
	}
	if len(inside) == 0 {
		return Pin{}, ErrNoCapital
	}
	return slices.MinFunc(inside, func(a, b place) int {
		return cmp.Or(cmp.Compare(a.rank, b.rank), cmp.Compare(b.pop, a.pop), strings.Compare(a.pin.Place, b.pin.Place))
	}).pin, nil
}

// As the map's capitals (apps/frontend/scripts/map/capitals.mjs): a country's main capital, then any other capital.
func rankOf(properties map[string]any) int {
	class, _ := properties["FEATURECLA"].(string)
	if class == "Admin-0 capital" && properties["ADM0CAP"] == 1.0 && properties["CAPALT"] == 0.0 {
		return 0
	}
	if strings.HasPrefix(class, "Admin-0 ") {
		return 1
	}
	return 2
}

func (p polygon) holds(longitude, latitude float64) bool {
	if len(p) == 0 || !inRing(p[0], longitude, latitude) {
		return false
	}
	return !slices.ContainsFunc(p[1:], func(hole [][]float64) bool { return inRing(hole, longitude, latitude) })
}

func inRing(ring [][]float64, x, y float64) bool {
	in := false
	for i, j := 0, len(ring)-1; i < len(ring); j, i = i, i+1 {
		xi, yi, xj, yj := ring[i][0], ring[i][1], ring[j][0], ring[j][1]
		if (yi > y) != (yj > y) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
			in = !in
		}
	}
	return in
}

package cpflags

import (
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"slices"
	"sync"

	flagdata "github.com/raphoester/clickplanet.lol-backend/generated/flags"
)

// BuiltInPin is the capital of a flag the game ships with that has no ground of its own: a region of a country.
type BuiltInPin struct {
	Code      string
	Latitude  float64
	Longitude float64
}

var pins = sync.OnceValue(func() []BuiltInPin {
	read, err := pinsOf(flagdata.Pins())
	if err != nil {
		panic(fmt.Errorf("the pins of the flags the game ships with: %w", err))
	}
	return read
})

// Pins is each region's capital, by code.
func Pins() []BuiltInPin {
	return slices.Clone(pins())
}

func pinsOf(file []byte) ([]BuiltInPin, error) {
	var kept map[string]struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	if err := json.Unmarshal(file, &kept); err != nil {
		return nil, fmt.Errorf("failed to read the pins: %w", err)
	}
	read := make([]BuiltInPin, 0, len(kept))
	for _, code := range slices.Sorted(maps.Keys(kept)) {
		pin := kept[code]
		if !IsBuiltIn(code) {
			return nil, fmt.Errorf("%q is no flag the game ships with", code)
		}
		if math.Abs(pin.Latitude) > 90 || math.Abs(pin.Longitude) > 180 {
			return nil, fmt.Errorf("%q is pinned off the globe: %v, %v", code, pin.Latitude, pin.Longitude)
		}
		read = append(read, BuiltInPin{Code: code, Latitude: pin.Latitude, Longitude: pin.Longitude})
	}
	return read, nil
}

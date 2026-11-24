package catalog

import (
	"errors"
	"fmt"
	"math"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
)

var (
	ErrInvalidPin = errors.New("a pin is a latitude from -90 to 90 and a longitude from -180 to 180, in degrees")
	// A call that names no pin is never one at 0, 0, in the sea off Africa.
	ErrNoPin = errors.New("a flag needs a pin")
)

// Pin is where a flag starts in Capital Siege, as a country starts on its capital: a point, never a tile, since each
// board numbers its tiles its own way.
type Pin struct {
	Latitude  float64
	Longitude float64
}

func ParsePin(latitude, longitude float64) (Pin, error) {
	pin := Pin{Latitude: latitude, Longitude: longitude}
	return pin, pin.check()
}

func (p Pin) check() error {
	if math.IsNaN(p.Latitude) || math.IsNaN(p.Longitude) || math.Abs(p.Latitude) > 90 || math.Abs(p.Longitude) > 180 {
		return fmt.Errorf("%w: %v, %v", ErrInvalidPin, p.Latitude, p.Longitude)
	}
	return nil
}

func (p Pin) message() *flagsv1.Pin {
	return &flagsv1.Pin{Latitude: p.Latitude, Longitude: p.Longitude}
}

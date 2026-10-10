package clicks

import "errors"

var (
	ErrNotWhole = errors.New("the flag does not hold the whole landmass")

	ErrFortifiedAlready = errors.New("the flag fortified the landmass last")
)

type Fortification struct {
	Landmass LandmassID
	// The country whose ground the landmass is.
	Ground  string
	Country string
	// The tile whose take made the landmass whole.
	Tile  uint32
	Tiles int
	// A tile already at the most it holds gains none.
	Raised []TileShields
}

func FortifyError(landmass LandmassID, flag string, held, size int, fortifiedBy string) error {
	switch {
	case landmass == NoLandmass || flag == "" || size == 0 || held < size:
		return ErrNotWhole
	case fortifiedBy == flag:
		return ErrFortifiedAlready
	}

	return nil
}

func SettlerOf(holder string, held, size int, fortifiedBy string) (string, bool) {
	if fortifiedBy != "" || holder == "" || size == 0 || held < size {
		return "", false
	}

	return holder, true
}

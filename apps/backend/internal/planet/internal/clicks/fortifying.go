package clicks

import "errors"

var (
	ErrNotWhole = errors.New("the flag does not hold the whole province")

	ErrFortifiedAlready = errors.New("the flag fortified the province last")
)

type Fortification struct {
	Province ProvinceID
	// The provinces file the province is a row of: a new cut numbers them again.
	Asset string
	// The country whose ground the province is.
	Ground  string
	Country string
	// The tile whose take made the province whole, and the account that took it.
	Tile    uint32
	Account string
	Tiles   int
	// A tile already at the most it holds gains none.
	Raised []TileShields
}

func FortifyError(province ProvinceID, flag string, held, size int, fortifiedBy string) error {
	switch {
	case province == NoProvince || flag == "" || size == 0 || held < size:
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

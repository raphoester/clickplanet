package clicks

import (
	"errors"
	"fmt"
)

const MaxAreaRings = 50

var ErrInvalidArea = errors.New("an area is 1 to 50 rings around a tile")

// An area is every tile within a number of tile spacings of one, as a bomb's blast is: what an operator restores or
// clears.
func NewAreas(geography *Geography, spacing float64) Areas {
	return Areas{geography: geography, spacing: spacing}
}

type Areas struct {
	geography *Geography
	spacing   float64
}

func (a Areas) Around(tile uint32, rings uint32) ([]uint32, error) {
	if rings == 0 || rings > MaxAreaRings {
		return nil, fmt.Errorf("%w: %d rings", ErrInvalidArea, rings)
	}

	centre, ok := a.geography.Position(tile)
	if !ok {
		return nil, fmt.Errorf("%w: %d", ErrTileOutOfRange, tile)
	}

	return a.geography.Within(centre, float64(rings)*a.spacing), nil
}

type Owners interface {
	Owner(tile uint32) (string, bool)
}

// Every tile of the area a flag holds, given to nobody against the owner it holds now.
func Emptying(area []uint32, owners Owners) []Restoration {
	var restorations []Restoration
	for _, tile := range area {
		if owner, _ := owners.Owner(tile); owner != "" {
			restorations = append(restorations, Restoration{Tile: tile, From: owner, To: ""})
		}
	}
	return restorations
}

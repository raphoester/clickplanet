package clicks

import "github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"

// MapCopy is the whole map as memory holds it: every owned tile, and each landmass a flag fortified.
type MapCopy struct {
	Tiles      []TileCopy
	Landmasses []LandmassCopy
}

type TileCopy struct {
	ID      uint32
	Owner   string
	Shields uint8
	// NoAccount when no player's take holds it.
	Holder cpsession.AccountID
}

// Asset is the borders blob the id indexes: a new map numbers its landmasses again.
type LandmassCopy struct {
	ID          LandmassID
	Asset       string
	FortifiedBy string
}

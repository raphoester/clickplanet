package clicks

import "github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"

// The tiles of a landmass one account's takes hold for one flag.
type Holding struct {
	Account cpsession.AccountID
	Flag    string
	Tiles   int
}

// One tile as the map holds it now: its flag, empty for nobody's, and the account whose take holds it,
// cpsession.NoAccount when no player's take does.
type TileHold struct {
	Tile   uint32
	Flag   string
	Holder cpsession.AccountID
}

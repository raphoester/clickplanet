package clicks

import "github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"

// The tiles of a landmass one account's takes hold for one flag.
type Holding struct {
	Account cpsession.AccountID
	Flag    string
	Tiles   int
}

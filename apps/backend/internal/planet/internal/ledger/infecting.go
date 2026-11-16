package ledger

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

const kindVirus = "virus"

// Infecting is one use or tick of a virus, on the tile it was used on: the tiles it took and the shields it struck.
type Infecting struct {
	Tile    uint32
	Scope   string
	Account string
	Country string
	At      time.Time
	Impacts []clicks.Impact
}

func (i Infecting) Replay(see func(Taking)) {
	replayClaims(see, i.Impacts, Taking{Scope: i.Scope, Account: i.Account, Country: i.Country, At: i.At})
}

func (i Infecting) Strikes(see func(Strike)) {
	strikeClaims(see, i.Impacts, Strike{Account: i.Account, Country: i.Country, At: i.At})
}

func (i Infecting) Show(see func(Scene)) {
	showImpacts(see, i.At, i.Country, i.Impacts)
}

func (i Infecting) Entry() (Entry, error) {
	return claimsEntry(Entry{
		Kind: kindVirus, Tile: i.Tile, Scope: i.Scope, Account: i.Account, Country: i.Country, At: i.At,
	}, i.Impacts)
}

func infectingOf(entry Entry) (Event, error) {
	impacts, err := claimsOf(entry)
	if err != nil {
		return nil, err
	}

	return Infecting{
		Tile: entry.Tile, Scope: entry.Scope, Account: entry.Account, Country: entry.Country, At: entry.At, Impacts: impacts,
	}, nil
}

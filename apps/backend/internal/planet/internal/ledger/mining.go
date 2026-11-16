package ledger

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

const kindMine = "mine"

// Mining is one mine gone off, on the tile it was laid on: the tiles its blast took for its flag and the shields it struck.
type Mining struct {
	Tile    uint32
	Scope   string
	Account string
	Country string
	At      time.Time
	Impacts []clicks.Impact
}

func (m Mining) Replay(see func(Taking)) {
	replayClaims(see, m.Impacts, Taking{Scope: m.Scope, Account: m.Account, Country: m.Country, At: m.At})
}

func (m Mining) Strikes(see func(Strike)) {
	strikeClaims(see, m.Impacts, Strike{Account: m.Account, Country: m.Country, At: m.At})
}

func (m Mining) Show(see func(Scene)) {
	showImpacts(see, m.At, m.Country, m.Impacts)
}

func (m Mining) Entry() (Entry, error) {
	return claimsEntry(Entry{
		Kind: kindMine, Tile: m.Tile, Scope: m.Scope, Account: m.Account, Country: m.Country, At: m.At,
	}, m.Impacts)
}

func miningOf(entry Entry) (Event, error) {
	impacts, err := claimsOf(entry)
	if err != nil {
		return nil, err
	}

	return Mining{
		Tile: entry.Tile, Scope: entry.Scope, Account: entry.Account, Country: entry.Country, At: entry.At, Impacts: impacts,
	}, nil
}

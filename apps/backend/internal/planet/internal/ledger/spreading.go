package ledger

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

const kindSpread = "spread"

type Spreading struct {
	Tile    uint32
	Scope   string
	Account string
	Country string
	At      time.Time
	Impacts []clicks.Impact
}

func (s Spreading) Replay(see func(Taking)) {
	replayClaims(see, s.Impacts, Taking{Scope: s.Scope, Account: s.Account, Country: s.Country, At: s.At})
}

func (s Spreading) Entry() (Entry, error) {
	return claimsEntry(Entry{
		Kind: kindSpread, Tile: s.Tile, Scope: s.Scope, Account: s.Account, Country: s.Country, At: s.At,
	}, s.Impacts)
}

func spreadingOf(entry Entry) (Event, error) {
	impacts, err := claimsOf(entry)
	if err != nil {
		return nil, err
	}

	return Spreading{
		Tile: entry.Tile, Scope: entry.Scope, Account: entry.Account, Country: entry.Country, At: entry.At, Impacts: impacts,
	}, nil
}

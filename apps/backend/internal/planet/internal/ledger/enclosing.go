package ledger

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

const kindEnclose = "enclose"

type Enclosing struct {
	Tile    uint32
	Scope   string
	Account string
	Country string
	At      time.Time
	Impacts []clicks.Impact
}

func (e Enclosing) Replay(see func(Taking)) {
	replayClaims(see, e.Impacts, Taking{Scope: e.Scope, Account: e.Account, Country: e.Country, At: e.At})
}

func (e Enclosing) Show(see func(Scene)) {
	showImpacts(see, e.At, e.Country, e.Impacts)
	see(Scene{At: e.At, Enclosure: bonusAt(e.Tile, e.Country, e.Impacts)})
}

func (e Enclosing) Entry() (Entry, error) {
	return claimsEntry(Entry{
		Kind: kindEnclose, Tile: e.Tile, Scope: e.Scope, Account: e.Account, Country: e.Country, At: e.At,
	}, e.Impacts)
}

func enclosingOf(entry Entry) (Event, error) {
	impacts, err := claimsOf(entry)
	if err != nil {
		return nil, err
	}

	return Enclosing{
		Tile: entry.Tile, Scope: entry.Scope, Account: entry.Account, Country: entry.Country, At: entry.At, Impacts: impacts,
	}, nil
}

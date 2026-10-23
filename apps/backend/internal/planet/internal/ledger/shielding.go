package ledger

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

const kindShield = "shield"

type Shielding struct {
	Tile    uint32
	Scope   string
	Account string
	Country string
	Shields int
	At      time.Time
}

func (s Shielding) Replay(func(Taking)) {}

func (s Shielding) Show(see func(Scene)) {
	see(Scene{At: s.At, Change: &Change{
		TileUpdate: clicks.TileUpdate{Tile: s.Tile, Value: s.Country, Previous: s.Country, Shields: s.Shields},
		Was:        s.Shields - 1,
	}})
}

func (s Shielding) Entry() (Entry, error) {
	payload, err := json.Marshal(shieldsPayload{Shields: s.Shields})
	if err != nil {
		return Entry{}, fmt.Errorf("failed to write down the shield on tile %d: %w", s.Tile, err)
	}

	return Entry{
		Kind: kindShield, Tile: s.Tile, Scope: s.Scope, Account: s.Account, Country: s.Country, Previous: s.Country, At: s.At,
		Payload: payload,
	}, nil
}

func shieldingOf(entry Entry) (Event, error) {
	var payload shieldsPayload
	if err := json.Unmarshal(entry.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to read the shield on tile %d: %w", entry.Tile, err)
	}

	return Shielding{
		Tile: entry.Tile, Scope: entry.Scope, Account: entry.Account, Country: entry.Country, Shields: payload.Shields,
		At: entry.At,
	}, nil
}

package ledger

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

const kindKnockOut = "knock_out"

// KnockingOut is a take that felled a flag's capital in an arena: every other tile the flag held went blank.
type KnockingOut struct {
	Tile    uint32
	Scope   string
	Account string
	// The flag whose take felled the capital, and the flag knocked out.
	Country string
	Out     string
	At      time.Time
	Cleared []uint32
}

func knockingOutOfImpact(by Caller, at time.Time, knockOut *clicks.KnockOut) KnockingOut {
	return KnockingOut{
		Tile: knockOut.Tile, Scope: by.Scope, Account: by.Account, Country: knockOut.By, Out: knockOut.Country, At: at,
		Cleared: knockOut.Cleared,
	}
}

// Replay gives each tile it cleared as a bomb does: a take for nobody.
func (k KnockingOut) Replay(see func(Taking)) {
	for _, tile := range k.Cleared {
		see(Taking{Tile: tile, Scope: k.Scope, Account: k.Account, Previous: k.Out, At: k.At})
	}
}

func (k KnockingOut) Strikes(func(Strike)) {}

func (k KnockingOut) Show(see func(Scene)) {
	for _, tile := range k.Cleared {
		see(Scene{At: k.At, Change: &Change{TileUpdate: clicks.TileUpdate{Tile: tile, Previous: k.Out}}})
	}
}

type knockOutPayload struct {
	Cleared []uint32 `json:"cleared"`
}

func (k KnockingOut) Entry() (Entry, error) {
	cleared := k.Cleared
	if cleared == nil {
		cleared = []uint32{}
	}
	payload, err := json.Marshal(knockOutPayload{Cleared: cleared})
	if err != nil {
		return Entry{}, fmt.Errorf("failed to write down the knock-out of %s: %w", k.Out, err)
	}

	return Entry{
		Kind: kindKnockOut, Tile: k.Tile, Scope: k.Scope, Account: k.Account, Country: k.Country, Previous: k.Out,
		At: k.At, Payload: payload,
	}, nil
}

func knockingOutOf(entry Entry) (Event, error) {
	var payload knockOutPayload
	if err := json.Unmarshal(entry.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to read the knock-out at tile %d: %w", entry.Tile, err)
	}

	return KnockingOut{
		Tile: entry.Tile, Scope: entry.Scope, Account: entry.Account, Country: entry.Country, Out: entry.Previous,
		At: entry.At, Cleared: payload.Cleared,
	}, nil
}

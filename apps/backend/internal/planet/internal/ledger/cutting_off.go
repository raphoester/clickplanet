package ledger

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

const kindCutOff = "cut_off"

// CuttingOff is a take that cut a flag's land from its capital: every tile of the land cut off went blank.
type CuttingOff struct {
	Tile    uint32
	Scope   string
	Account string
	// The flag whose take cut the land, and the flag cut off.
	Country string
	Cut     string
	At      time.Time
	Cleared []uint32
}

func cuttingOffOfImpact(by Caller, at time.Time, cutOff *clicks.CutOff) CuttingOff {
	return CuttingOff{
		Tile: cutOff.Tile, Scope: by.Scope, Account: by.Account, Country: cutOff.By, Cut: cutOff.Country, At: at,
		Cleared: cutOff.Cleared,
	}
}

// Replay gives each tile it cleared as a bomb does: a take for nobody.
func (c CuttingOff) Replay(see func(Taking)) {
	for _, tile := range c.Cleared {
		see(Taking{Tile: tile, Scope: c.Scope, Account: c.Account, Previous: c.Cut, At: c.At})
	}
}

func (c CuttingOff) Strikes(func(Strike)) {}

func (c CuttingOff) Show(see func(Scene)) {
	for _, tile := range c.Cleared {
		see(Scene{At: c.At, Change: &Change{TileUpdate: clicks.TileUpdate{Tile: tile, Previous: c.Cut}}})
	}
}

func (c CuttingOff) Entry() (Entry, error) {
	cleared := c.Cleared
	if cleared == nil {
		cleared = []uint32{}
	}
	payload, err := json.Marshal(clearedPayload{Cleared: cleared})
	if err != nil {
		return Entry{}, fmt.Errorf("failed to write down the cut-off of %s: %w", c.Cut, err)
	}

	return Entry{
		Kind: kindCutOff, Tile: c.Tile, Scope: c.Scope, Account: c.Account, Country: c.Country, Previous: c.Cut,
		At: c.At, Payload: payload,
	}, nil
}

func cuttingOffOf(entry Entry) (Event, error) {
	var payload clearedPayload
	if err := json.Unmarshal(entry.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to read the cut-off at tile %d: %w", entry.Tile, err)
	}

	return CuttingOff{
		Tile: entry.Tile, Scope: entry.Scope, Account: entry.Account, Country: entry.Country, Cut: entry.Previous,
		At: entry.At, Cleared: payload.Cleared,
	}, nil
}

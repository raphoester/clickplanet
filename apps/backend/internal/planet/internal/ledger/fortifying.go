package ledger

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

const kindFortify = "fortify"

type Fortifying struct {
	Tile     uint32
	Landmass clicks.LandmassID
	Scope    string
	Account  string
	Country  string
	At       time.Time
	Raised   []clicks.TileShields
}

func fortifyingOfImpact(by Caller, at time.Time, flag string, fortified *clicks.Fortification) Fortifying {
	return Fortifying{
		Tile: fortified.Tile, Landmass: fortified.Landmass, Scope: by.Scope, Account: by.Account, Country: flag, At: at,
		Raised: fortified.Raised,
	}
}

func (f Fortifying) Replay(func(Taking)) {}

func (f Fortifying) Show(see func(Scene)) {
	for _, raised := range f.Raised {
		see(Scene{At: f.At, Change: &Change{
			TileUpdate: clicks.TileUpdate{Tile: raised.Tile, Value: f.Country, Previous: f.Country, Shields: raised.Shields},
			Was:        raised.Shields - 1,
		}})
	}
}

type raisedJSON struct {
	Tile    uint32 `json:"tile"`
	Shields int    `json:"shields"`
}

type fortifyPayload struct {
	Landmass clicks.LandmassID `json:"landmass"`
	Raised   []raisedJSON      `json:"raised"`
}

func (f Fortifying) Entry() (Entry, error) {
	raised := make([]raisedJSON, len(f.Raised))
	for i, tile := range f.Raised {
		raised[i] = raisedJSON{Tile: tile.Tile, Shields: tile.Shields}
	}

	payload, err := json.Marshal(fortifyPayload{Landmass: f.Landmass, Raised: raised})
	if err != nil {
		return Entry{}, fmt.Errorf("failed to write down the fortify of landmass %d: %w", f.Landmass, err)
	}

	return Entry{
		Kind: kindFortify, Tile: f.Tile, Scope: f.Scope, Account: f.Account, Country: f.Country, Previous: f.Country,
		At: f.At, Payload: payload,
	}, nil
}

func fortifyingOf(entry Entry) (Event, error) {
	var payload fortifyPayload
	if err := json.Unmarshal(entry.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to read the fortify at tile %d: %w", entry.Tile, err)
	}

	raised := make([]clicks.TileShields, len(payload.Raised))
	for i, tile := range payload.Raised {
		raised[i] = clicks.TileShields{Tile: tile.Tile, Shields: tile.Shields}
	}

	return Fortifying{
		Tile: entry.Tile, Landmass: payload.Landmass, Scope: entry.Scope, Account: entry.Account, Country: entry.Country,
		At: entry.At, Raised: raised,
	}, nil
}

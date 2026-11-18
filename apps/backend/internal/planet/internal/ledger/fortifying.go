package ledger

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

const kindFortify = "fortify"

// A fortify names its province and the provinces file it is a row of. One made before fortify played on provinces
// (2026-11-18) names its landmass instead, a row of its board's borders, and no province.
type Fortifying struct {
	Tile      uint32
	Province  clicks.ProvinceID
	Provinces string
	Landmass  clicks.LandmassID
	Ground    string
	Tiles     int
	Scope     string
	Account   string
	Country   string
	At        time.Time
	Raised    []clicks.TileShields
}

func fortifyingOfImpact(by Caller, at time.Time, flag string, fortified *clicks.Fortification) Fortifying {
	return Fortifying{
		Tile: fortified.Tile, Province: fortified.Province, Provinces: fortified.Asset, Ground: fortified.Ground,
		Tiles: fortified.Tiles, Scope: by.Scope, Account: by.Account, Country: flag, At: at, Raised: fortified.Raised,
	}
}

func (f Fortifying) Replay(func(Taking)) {}

func (f Fortifying) Strikes(func(Strike)) {}

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
	Province  clicks.ProvinceID `json:"province,omitempty"`
	Provinces string            `json:"provinces,omitempty"`
	Landmass  clicks.LandmassID `json:"landmass,omitempty"`
	Ground    string            `json:"ground,omitempty"`
	Tiles     int               `json:"tiles,omitempty"`
	Raised    []raisedJSON      `json:"raised"`
}

func (f Fortifying) Entry() (Entry, error) {
	raised := make([]raisedJSON, len(f.Raised))
	for i, tile := range f.Raised {
		raised[i] = raisedJSON{Tile: tile.Tile, Shields: tile.Shields}
	}

	payload, err := json.Marshal(fortifyPayload{
		Province: f.Province, Provinces: f.Provinces, Landmass: f.Landmass, Ground: f.Ground, Tiles: f.Tiles, Raised: raised,
	})
	if err != nil {
		return Entry{}, fmt.Errorf("failed to write down the fortify of province %d: %w", f.Province, err)
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
		Tile: entry.Tile, Province: payload.Province, Provinces: payload.Provinces, Landmass: payload.Landmass,
		Ground: payload.Ground, Tiles: payload.Tiles, Scope: entry.Scope, Account: entry.Account, Country: entry.Country,
		At: entry.At, Raised: raised,
	}, nil
}

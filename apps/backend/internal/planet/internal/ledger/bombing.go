package ledger

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

const kindBomb = "bomb"

type Bombing struct {
	Scope   string
	Account string
	At      time.Time
	Blast   clicks.Blast
}

type bombPayload struct {
	Flag    string       `json:"flag"`
	Point   [3]float64   `json:"point"`
	Radius  float64      `json:"radius"`
	Cleared []changeJSON `json:"cleared,omitempty"`
	Struck  []strikeJSON `json:"struck,omitempty"`
}

func (b Bombing) Replay(see func(Taking)) {
	for i, tile := range b.Blast.Cleared {
		see(Taking{Tile: tile, Scope: b.Scope, Account: b.Account, Previous: b.Blast.Owners[i], At: b.At})
	}
}

func (b Bombing) Entry() (Entry, error) {
	payload := bombPayload{
		Flag:   b.Blast.CountryID,
		Point:  [3]float64{b.Blast.Point.X, b.Blast.Point.Y, b.Blast.Point.Z},
		Radius: b.Blast.Radius,
	}

	entry := Entry{Kind: kindBomb, Tile: b.Blast.Tile, Scope: b.Scope, Account: b.Account, At: b.At}
	for i, tile := range b.Blast.Cleared {
		payload.Cleared = append(payload.Cleared, changeJSON{Tile: tile, Owner: b.Blast.Owners[i]})
		if tile == b.Blast.Tile {
			entry.Previous = b.Blast.Owners[i]
		}
	}
	for i, tile := range b.Blast.Struck {
		payload.Struck = append(payload.Struck, strikeJSON{Tile: tile, Shields: b.Blast.Left[i]})
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return Entry{}, fmt.Errorf("failed to write down the bomb at tile %d: %w", b.Blast.Tile, err)
	}
	entry.Payload = encoded

	return entry, nil
}

func bombingOf(entry Entry) (Event, error) {
	var payload bombPayload
	if err := json.Unmarshal(entry.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to read the bomb at tile %d: %w", entry.Tile, err)
	}

	blast := clicks.Blast{
		Tile:      entry.Tile,
		CountryID: payload.Flag,
		Point:     clicks.Vec3{X: payload.Point[0], Y: payload.Point[1], Z: payload.Point[2]},
		Radius:    payload.Radius,
	}
	for _, cleared := range payload.Cleared {
		blast.Cleared = append(blast.Cleared, cleared.Tile)
		blast.Owners = append(blast.Owners, cleared.Owner)
	}
	for _, struck := range payload.Struck {
		blast.Struck = append(blast.Struck, struck.Tile)
		blast.Left = append(blast.Left, struck.Shields)
	}

	return Bombing{Scope: entry.Scope, Account: entry.Account, At: entry.At, Blast: blast}, nil
}

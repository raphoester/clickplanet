package ledger

import (
	"encoding/json"
	"fmt"
	"time"
)

const kindStrike = "strike"

type Striking struct {
	Tile    uint32
	Scope   string
	Account string
	Country string
	Owner   string
	Shields int
	At      time.Time
}

type shieldsPayload struct {
	Shields int `json:"shields"`
}

func (s Striking) Replay(func(Taking)) {}

func (s Striking) Show(see func(Scene)) {
	see(Scene{At: s.At, Change: struck(s.Tile, s.Owner, s.Shields, true)})
}

func (s Striking) Entry() (Entry, error) {
	payload, err := json.Marshal(shieldsPayload{Shields: s.Shields})
	if err != nil {
		return Entry{}, fmt.Errorf("failed to write down the strike on tile %d: %w", s.Tile, err)
	}

	return Entry{
		Kind: kindStrike, Tile: s.Tile, Scope: s.Scope, Account: s.Account, Country: s.Country, Previous: s.Owner, At: s.At,
		Payload: payload,
	}, nil
}

func strikingOf(entry Entry) (Event, error) {
	var payload shieldsPayload
	if err := json.Unmarshal(entry.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to read the strike on tile %d: %w", entry.Tile, err)
	}

	return Striking{
		Tile: entry.Tile, Scope: entry.Scope, Account: entry.Account, Country: entry.Country, Owner: entry.Previous,
		Shields: payload.Shields, At: entry.At,
	}, nil
}

package ledger

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type changeJSON struct {
	Tile  uint32 `json:"tile"`
	Owner string `json:"owner,omitempty"`
}

type strikeJSON struct {
	Tile    uint32 `json:"tile"`
	Owner   string `json:"owner,omitempty"`
	Shields int    `json:"shields"`
}

type claimsPayload struct {
	Taken  []changeJSON `json:"taken,omitempty"`
	Struck []strikeJSON `json:"struck,omitempty"`
}

func changed(impacts []clicks.Impact) []clicks.Impact {
	var kept []clicks.Impact
	for _, impact := range impacts {
		if impact.Outcome != clicks.Unchanged {
			kept = append(kept, impact)
		}
	}
	return byTile(kept)
}

func byTile(impacts []clicks.Impact) []clicks.Impact {
	slices.SortFunc(impacts, func(a, b clicks.Impact) int { return cmp.Compare(a.Tile, b.Tile) })
	return impacts
}

func replayClaims(see func(Taking), impacts []clicks.Impact, by Taking) {
	for _, impact := range impacts {
		if impact.Outcome == clicks.Taken {
			by.Tile, by.Previous = impact.Tile, impact.Owner
			see(by)
		}
	}
}

func claimsEntry(entry Entry, impacts []clicks.Impact) (Entry, error) {
	var payload claimsPayload
	for _, impact := range impacts {
		switch impact.Outcome {
		case clicks.Taken:
			payload.Taken = append(payload.Taken, changeJSON{Tile: impact.Tile, Owner: impact.Owner})
		case clicks.Shielded:
			payload.Struck = append(payload.Struck, strikeJSON{Tile: impact.Tile, Owner: impact.Owner, Shields: impact.Shields})
		case clicks.Unchanged:
		}
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return Entry{}, fmt.Errorf("failed to write down the %s at tile %d: %w", entry.Kind, entry.Tile, err)
	}
	entry.Payload = encoded

	return entry, nil
}

func claimsOf(entry Entry) ([]clicks.Impact, error) {
	var payload claimsPayload
	if err := json.Unmarshal(entry.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to read the %s at tile %d: %w", entry.Kind, entry.Tile, err)
	}

	var impacts []clicks.Impact
	for _, taken := range payload.Taken {
		impacts = append(impacts, clicks.Impact{Tile: taken.Tile, Owner: taken.Owner, Outcome: clicks.Taken})
	}
	for _, struck := range payload.Struck {
		impacts = append(impacts, clicks.Impact{
			Tile: struck.Tile, Owner: struck.Owner, Outcome: clicks.Shielded, Shields: struck.Shields,
		})
	}

	return byTile(impacts), nil
}

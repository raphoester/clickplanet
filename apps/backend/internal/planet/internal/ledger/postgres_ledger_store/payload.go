package postgres_ledger_store

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

const (
	kindTake = "take"
	kindBomb = "bomb"
)

type bombPayload struct {
	Flag    string              `json:"flag"`
	Point   [3]float64          `json:"point"`
	Radius  float64             `json:"radius"`
	Cleared map[string][]uint32 `json:"cleared"`
}

func payloadOf(blast clicks.Blast) (string, error) {
	payload := bombPayload{
		Flag:    blast.CountryID,
		Point:   [3]float64{blast.Point.X, blast.Point.Y, blast.Point.Z},
		Radius:  blast.Radius,
		Cleared: make(map[string][]uint32),
	}
	for i, tile := range blast.Cleared {
		payload.Cleared[blast.Owners[i]] = append(payload.Cleared[blast.Owners[i]], tile)
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to encode a bomb: %w", err)
	}
	return string(encoded), nil
}

func blastOf(tile uint32, encoded []byte) (clicks.Blast, error) {
	var payload bombPayload
	if err := json.Unmarshal(encoded, &payload); err != nil {
		return clicks.Blast{}, fmt.Errorf("failed to decode the bomb at tile %d: %w", tile, err)
	}

	type clearing struct {
		tile  uint32
		owner string
	}
	var clearings []clearing
	for owner, tiles := range payload.Cleared {
		for _, cleared := range tiles {
			clearings = append(clearings, clearing{tile: cleared, owner: owner})
		}
	}
	slices.SortFunc(clearings, func(a, b clearing) int { return cmp.Compare(a.tile, b.tile) })

	blast := clicks.Blast{
		Tile:      tile,
		CountryID: payload.Flag,
		Point:     clicks.Vec3{X: payload.Point[0], Y: payload.Point[1], Z: payload.Point[2]},
		Radius:    payload.Radius,
	}
	for _, c := range clearings {
		blast.Cleared = append(blast.Cleared, c.tile)
		blast.Owners = append(blast.Owners, c.owner)
	}

	return blast, nil
}

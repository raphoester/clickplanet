package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"math/rand/v2"
	"os"
	"path/filepath"
	"time"
)

// seed is what the old backend's data files hold before any traffic, and what must still be there after.
type seed struct {
	Owners     map[uint32]string `json:"owners"`
	ChatIDs    []string          `json:"chatIds"`
	BanScopes  []string          `json:"banScopes"`
	OwnedTiles int               `json:"ownedTiles"`
}

var countries = []string{"fr", "us", "de", "br", "jp", "in", "ng", "mx"}

// writeSeed fills dir with tiles.snapshot, bans.jsonl and chat.log, and records what they hold in seed.json.
func writeSeed(dir string, maxIndex uint32, ownedShare float64) error {
	rng := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // a fixed seed makes every run the same map.

	s := seed{Owners: make(map[uint32]string)}
	for tile := uint32(0); tile <= maxIndex; tile++ {
		if rng.Float64() < ownedShare {
			s.Owners[tile] = countries[rng.IntN(len(countries))]
		}
	}
	s.OwnedTiles = len(s.Owners)

	if err := os.WriteFile(filepath.Join(dir, "tiles.snapshot"), encodeSnapshot(maxIndex, s.Owners), 0o600); err != nil {
		return fmt.Errorf("write tiles.snapshot: %w", err)
	}

	bans, scopes := seedBans()
	s.BanScopes = scopes
	if err := os.WriteFile(filepath.Join(dir, "bans.jsonl"), bans, 0o600); err != nil {
		return fmt.Errorf("write bans.jsonl: %w", err)
	}

	chat, ids := seedChat()
	s.ChatIDs = ids
	if err := os.WriteFile(filepath.Join(dir, "chat.log"), chat, 0o600); err != nil {
		return fmt.Errorf("write chat.log: %w", err)
	}

	raw, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("encode seed: %w", err)
	}

	return os.WriteFile(filepath.Join(dir, "seed.json"), raw, 0o600)
}

// encodeSnapshot is the tile snapshot format the backend used before postgres: magic, version, CRC32, code table, two bytes per tile.
func encodeSnapshot(maxIndex uint32, owners map[uint32]string) []byte {
	codes := []string{""}
	ids := map[string]uint16{"": 0}
	tiles := make([]uint16, maxIndex+1)
	for tile, owner := range owners {
		id, ok := ids[owner]
		if !ok {
			id = uint16(len(codes)) //nolint:gosec // a handful of countries.
			ids[owner] = id
			codes = append(codes, owner)
		}
		tiles[tile] = id
	}

	payload := binary.LittleEndian.AppendUint32(make([]byte, 0, 8+2*len(tiles)), maxIndex)
	payload = binary.LittleEndian.AppendUint16(payload, uint16(len(codes))) //nolint:gosec // a handful of countries.
	for _, code := range codes {
		payload = append(payload, uint8(len(code))) //nolint:gosec // two-letter codes.
		payload = append(payload, code...)
	}
	for _, tile := range tiles {
		payload = binary.LittleEndian.AppendUint16(payload, tile)
	}

	raw := append([]byte("CPTILES\n"), 1)
	raw = binary.LittleEndian.AppendUint32(raw, crc32.ChecksumIEEE(payload))

	return append(raw, payload...)
}

func seedBans() ([]byte, []string) {
	scopes := []string{"192.0.2.10", "192.0.2.11"}
	until := time.Now().Add(365 * 24 * time.Hour).UTC()

	var lines []byte
	for _, scope := range scopes {
		line, _ := json.Marshal(map[string]any{"scope": scope, "flags": 1, "offences": 1, "until": until})
		lines = append(append(lines, line...), '\n')
	}

	return lines, scopes
}

func seedChat() ([]byte, []string) {
	ids := []string{"e2e-chat-1", "e2e-chat-2", "e2e-chat-3"}
	at := time.Now().Add(-time.Hour).UTC()

	var lines []byte
	for i, id := range ids {
		line, _ := json.Marshal(map[string]any{
			"at": at.Add(time.Duration(i) * time.Minute), "id": id, "name": "seed", "tag": "abcdef",
			"country": "fr", "ip": "192.0.2.20", "text": "seeded before the rollout",
		})
		lines = append(append(lines, line...), '\n')
	}

	return lines, ids
}

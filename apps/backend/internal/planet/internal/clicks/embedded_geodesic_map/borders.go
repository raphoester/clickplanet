package embedded_geodesic_map

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	mapdata "github.com/raphoester/clickplanet.lol-backend/generated/map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// LoadBorders reads the tile to landmass table the frontend's `npm run borders` writes.
// Format: uint32 header length | JSON {tiles, codes} | tiles*2 uint16 landmass | frames and totals, unread here.
func (l *Loader) LoadBorders() (*clicks.Borders, error) {
	blob, asset, err := mapdata.Borders()
	if err != nil {
		return nil, fmt.Errorf("failed to read the embedded borders blob: %w", err)
	}

	borders, err := decodeBorders(blob)
	if err != nil {
		return nil, fmt.Errorf("failed to decode %s: %w", asset, err)
	}

	if borders.Tiles() != l.expectTiles {
		return nil, fmt.Errorf(
			"%s holds %d tiles but gameMap.maxIndex is %d: the blob and the config were updated apart",
			asset, borders.Tiles(), l.expectTiles)
	}

	l.logger.Info("map borders loaded", slog.String("asset", asset))

	return borders, nil
}

func decodeBorders(blob []byte) (*clicks.Borders, error) {
	if len(blob) < 4 {
		return nil, errors.New("too short for a header length")
	}

	headerEnd := 4 + int(binary.LittleEndian.Uint32(blob))
	if headerEnd > len(blob) {
		return nil, fmt.Errorf("header of %d bytes runs past the %d-byte blob", headerEnd-4, len(blob))
	}

	var header struct {
		Tiles int      `json:"tiles"`
		Codes []string `json:"codes"`
	}
	if err := json.Unmarshal(blob[4:headerEnd], &header); err != nil {
		return nil, fmt.Errorf("failed to parse the header: %w", err)
	}

	if header.Tiles < 0 || headerEnd+header.Tiles*2 > len(blob) {
		return nil, fmt.Errorf("%d tiles run past the %d-byte blob", header.Tiles, len(blob))
	}

	regions := make([]uint16, header.Tiles)
	for i := range regions {
		regions[i] = binary.LittleEndian.Uint16(blob[headerEnd+i*2:])
	}

	borders, err := clicks.NewBorders(regions, header.Codes)
	if err != nil {
		return nil, fmt.Errorf("failed to build the borders: %w", err)
	}

	return borders, nil
}

package embedded_board

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpboards"
)

func loadBorders(board cpboards.Board) (*clicks.Borders, error) {
	asset := board.Manifest.Files.Borders
	blob, err := board.File(asset)
	if err != nil {
		return nil, err //nolint:wrapcheck // the board names the file.
	}

	borders, err := decodeBorders(asset, blob)
	if err != nil {
		return nil, fmt.Errorf("failed to decode %s: %w", asset, err)
	}
	if err := checkTiles(board, asset, borders.Tiles()); err != nil {
		return nil, err
	}

	return borders, nil
}

func decodeBorders(asset string, blob []byte) (*clicks.Borders, error) {
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

	borders, err := clicks.NewBorders(asset, regions, header.Codes)
	if err != nil {
		return nil, fmt.Errorf("failed to build the borders: %w", err)
	}

	return borders, nil
}

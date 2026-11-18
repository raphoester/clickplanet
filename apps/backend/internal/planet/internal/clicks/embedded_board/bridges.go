package embedded_board

import (
	"encoding/json"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpboards"
)

func loadBridges(board cpboards.Board) (*clicks.Bridges, error) {
	asset := board.Manifest.Files.Bridges
	blob, err := board.File(asset)
	if err != nil {
		return nil, err //nolint:wrapcheck // the board names the file.
	}

	var file struct {
		Tiles   uint32      `json:"tiles"`
		Bridges [][2]uint32 `json:"bridges"`
	}
	if err := json.Unmarshal(blob, &file); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", asset, err)
	}
	if err := checkTiles(board, asset, file.Tiles); err != nil {
		return nil, err
	}

	bridges, err := clicks.NewBridges(file.Tiles, file.Bridges)
	if err != nil {
		return nil, fmt.Errorf("failed to build the bridges from %s: %w", asset, err)
	}

	return bridges, nil
}

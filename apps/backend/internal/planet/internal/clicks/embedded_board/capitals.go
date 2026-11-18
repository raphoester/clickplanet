package embedded_board

import (
	"encoding/json"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpboards"
)

// loadCapitals refuses a capital off its own country's ground, which a file of another board would put there.
func loadCapitals(board cpboards.Board, borders *clicks.Borders) (*clicks.Capitals, error) {
	asset := board.Manifest.Files.Capitals
	blob, err := board.File(asset)
	if err != nil {
		return nil, err //nolint:wrapcheck // the board names the file.
	}

	var file struct {
		Tiles    uint32            `json:"tiles"`
		Capitals map[string]uint32 `json:"capitals"`
	}
	if err := json.Unmarshal(blob, &file); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", asset, err)
	}
	if err := checkTiles(board, asset, file.Tiles); err != nil {
		return nil, err
	}
	for country, tile := range file.Capitals {
		if ground := borders.CountryOf(tile); ground != country {
			return nil, fmt.Errorf("%s puts %s's capital on tile %d, which is %q's ground", asset, country, tile, ground)
		}
	}

	return clicks.NewCapitals(file.Capitals), nil
}

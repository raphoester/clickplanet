package embedded_board

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// LoadCapitals refuses a capital off its own country's ground, which a file of another board would put there.
func (l *Loader) LoadCapitals(borders *clicks.Borders) (*clicks.Capitals, error) {
	board, err := l.board()
	if err != nil {
		return nil, err
	}
	asset := board.manifest.Files.Capitals
	blob, err := board.file(asset)
	if err != nil {
		return nil, err
	}

	var file struct {
		Tiles    uint32            `json:"tiles"`
		Capitals map[string]uint32 `json:"capitals"`
	}
	if err := json.Unmarshal(blob, &file); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", asset, err)
	}
	if file.Tiles != l.expectTiles {
		return nil, fmt.Errorf(
			"%s holds %d tiles but gameMap.maxIndex is %d: the board and the config were updated apart",
			asset, file.Tiles, l.expectTiles)
	}
	for country, tile := range file.Capitals {
		if ground := borders.CountryOf(tile); ground != country {
			return nil, fmt.Errorf("%s puts %s's capital on tile %d, which is %q's ground", asset, country, tile, ground)
		}
	}

	l.logger.Info("map capitals loaded", slog.String("asset", asset), slog.Int("capitals", len(file.Capitals)))

	return clicks.NewCapitals(file.Capitals), nil
}

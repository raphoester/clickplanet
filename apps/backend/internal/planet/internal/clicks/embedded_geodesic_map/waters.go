package embedded_geodesic_map

import (
	"encoding/json"
	"fmt"
	"log/slog"

	mapdata "github.com/raphoester/clickplanet.lol-backend/generated/map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func (l *Loader) LoadWaters() (*clicks.Waters, error) {
	blob, asset, err := mapdata.Waters()
	if err != nil {
		return nil, fmt.Errorf("failed to read the embedded waters blob: %w", err)
	}

	var file struct {
		Tiles  uint32     `json:"tiles"`
		Bodies [][]uint32 `json:"bodies"`
	}
	if err := json.Unmarshal(blob, &file); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", asset, err)
	}
	if file.Tiles != l.expectTiles {
		return nil, fmt.Errorf(
			"%s holds %d tiles but gameMap.maxIndex is %d: the blob and the config were updated apart",
			asset, file.Tiles, l.expectTiles)
	}

	waters, err := clicks.NewWaters(file.Tiles, file.Bodies)
	if err != nil {
		return nil, fmt.Errorf("failed to build the waters from %s: %w", asset, err)
	}

	l.logger.Info("map waters loaded", slog.String("asset", asset), slog.Int("bodies", waters.Bodies()))

	return waters, nil
}

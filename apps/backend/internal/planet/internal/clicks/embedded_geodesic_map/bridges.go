package embedded_geodesic_map

import (
	"encoding/json"
	"fmt"
	"log/slog"

	mapdata "github.com/raphoester/clickplanet.lol-backend/generated/map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func (l *Loader) LoadBridges() (*clicks.Bridges, error) {
	blob, asset, err := mapdata.Bridges()
	if err != nil {
		return nil, fmt.Errorf("failed to read the embedded bridges blob: %w", err)
	}

	var file struct {
		Tiles   uint32      `json:"tiles"`
		Bridges [][2]uint32 `json:"bridges"`
	}
	if err := json.Unmarshal(blob, &file); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", asset, err)
	}
	if file.Tiles != l.expectTiles {
		return nil, fmt.Errorf(
			"%s holds %d tiles but gameMap.maxIndex is %d: the blob and the config were updated apart",
			asset, file.Tiles, l.expectTiles)
	}

	bridges, err := clicks.NewBridges(file.Tiles, file.Bridges)
	if err != nil {
		return nil, fmt.Errorf("failed to build the bridges from %s: %w", asset, err)
	}

	l.logger.Info("map bridges loaded", slog.String("asset", asset), slog.Int("bridges", bridges.Count()))

	return bridges, nil
}

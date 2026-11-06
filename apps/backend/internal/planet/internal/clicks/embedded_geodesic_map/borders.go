package embedded_geodesic_map

import (
	"fmt"
	"log/slog"

	mapdata "github.com/raphoester/clickplanet.lol-backend/generated/map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func (l *Loader) LoadBorders() (*clicks.Borders, error) {
	blob, asset, err := mapdata.Borders()
	if err != nil {
		return nil, fmt.Errorf("failed to read the embedded borders blob: %w", err)
	}

	borders, err := decodeBorders(asset, blob)
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

func decodeBorders(asset string, blob []byte) (*clicks.Borders, error) {
	regions, codes, err := mapdata.DecodeBorders(blob)
	if err != nil {
		return nil, err //nolint:wrapcheck // LoadBorders names the asset.
	}

	borders, err := clicks.NewBorders(asset, regions, codes)
	if err != nil {
		return nil, fmt.Errorf("failed to build the borders: %w", err)
	}

	return borders, nil
}

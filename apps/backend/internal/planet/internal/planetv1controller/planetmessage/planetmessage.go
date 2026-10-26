package planetmessage

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func Version(version clicks.MapVersion) *planetv1.MapVersion {
	if version == (clicks.MapVersion{}) {
		return nil
	}
	return &planetv1.MapVersion{Epoch: version.Epoch, Seq: version.Seq}
}

func TileUpdate(update clicks.TileUpdate, version clicks.MapVersion) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_TileUpdate{TileUpdate: &planetv1.TileUpdate{
			TileId:            update.Tile,
			CountryId:         update.Value,
			PreviousCountryId: update.Previous,
			Clicked:           update.Clicked,
			Shields:           uint32(update.Shields), //nolint:gosec // a byte in the tile storage.
			Version:           Version(version),
		}},
	}
}

func BombDropped(blast *clicks.Blast, version clicks.MapVersion) *planetv1.PlanetEvent {
	struck := make([]*planetv1.TileShields, len(blast.Struck))
	for i, tile := range blast.Struck {
		struck[i] = &planetv1.TileShields{TileId: tile, Shields: uint32(blast.Left[i])} //nolint:gosec // a byte in the tile storage.
	}

	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_BombDropped{
			BombDropped: &planetv1.BombDropped{
				TileId:         blast.Tile,
				CountryId:      blast.CountryID,
				Radius:         blast.Radius,
				ClearedTileIds: blast.Cleared,
				StruckTileIds:  blast.Struck, //nolint:staticcheck // the tabs loaded before it was deprecated still read it.
				Point:          &planetv1.GlobePoint{X: blast.Point.X, Y: blast.Point.Y, Z: blast.Point.Z},
				Version:        Version(version),
				Struck:         struck,
			},
		},
	}
}

func LandmassFortified(fortification *clicks.Fortification, version clicks.MapVersion) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_LandmassFortified{
			LandmassFortified: &planetv1.LandmassFortified{
				LandmassId: uint32(fortification.Landmass),
				CountryId:  fortification.Country,
				TileId:     fortification.Tile,
				Version:    Version(version),
			},
		},
	}
}

func HoldingsChanged(tally *clicks.Tally, version clicks.MapVersion) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_HoldingsChanged{
			HoldingsChanged: &planetv1.HoldingsChanged{
				Version:    Version(version),
				Countries:  Holdings(tally.Countries),
				Landmasses: LandmassHoldings(tally.Landmasses),
			},
		},
	}
}

func Holdings(counts []clicks.FlagTiles) []*planetv1.Holding {
	holdings := make([]*planetv1.Holding, len(counts))
	for i, count := range counts {
		holdings[i] = &planetv1.Holding{CountryId: count.Flag, Tiles: count.Tiles}
	}
	return holdings
}

func LandmassHoldings(rows []clicks.LandmassTiles) []*planetv1.LandmassHoldings {
	landmasses := make([]*planetv1.LandmassHoldings, len(rows))
	for i, row := range rows {
		landmasses[i] = &planetv1.LandmassHoldings{LandmassId: uint32(row.Landmass), Holdings: Holdings(row.Flags)}
	}
	return landmasses
}

func TilesSpread(country string, tile uint32, spread []uint32) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_TilesSpread{
			TilesSpread: &planetv1.TilesSpread{CountryId: country, TileId: tile, SpreadTileIds: spread},
		},
	}
}

func TilesEnclosed(country string, closing uint32, wall, filled []uint32, yours bool) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_TilesEnclosed{
			TilesEnclosed: &planetv1.TilesEnclosed{
				CountryId: country, ClosingTileId: closing, WallTileIds: wall, FilledTileIds: filled, Yours: yours,
			},
		},
	}
}

func Map(batch clicks.DenseBatch) *planetv1.GetMapResponse {
	shields := make([]*planetv1.TileShields, len(batch.Shields))
	for i, tile := range batch.Shields {
		shields[i] = &planetv1.TileShields{
			TileId:  tile.Tile,
			Shields: uint32(tile.Shields), //nolint:gosec // a byte in the tile storage.
		}
	}

	return &planetv1.GetMapResponse{
		StartTileId: batch.Start,
		Codes:       batch.Codes,
		Tiles:       batch.Tiles,
		Shields:     shields,
		Version:     Version(batch.Version),
	}
}

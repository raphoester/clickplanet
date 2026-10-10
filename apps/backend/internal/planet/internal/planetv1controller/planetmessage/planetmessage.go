package planetmessage

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func TileUpdate(update clicks.TileUpdate) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_TileUpdate{TileUpdate: &planetv1.TileUpdate{
			TileId:            update.Tile,
			CountryId:         update.Value,
			PreviousCountryId: update.Previous,
			Clicked:           update.Clicked,
			Shields:           uint32(update.Shields), //nolint:gosec // a byte in the tile storage.
		}},
	}
}

func BombDropped(blast *clicks.Blast) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_BombDropped{
			BombDropped: &planetv1.BombDropped{
				TileId:         blast.Tile,
				CountryId:      blast.CountryID,
				Radius:         blast.Radius,
				ClearedTileIds: blast.Cleared,
				StruckTileIds:  blast.Struck,
				Point:          &planetv1.GlobePoint{X: blast.Point.X, Y: blast.Point.Y, Z: blast.Point.Z},
			},
		},
	}
}

func LandmassFortified(fortification *clicks.Fortification) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_LandmassFortified{
			LandmassFortified: &planetv1.LandmassFortified{
				LandmassId: uint32(fortification.Landmass),
				CountryId:  fortification.Country,
				TileId:     fortification.Tile,
			},
		},
	}
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
	}
}

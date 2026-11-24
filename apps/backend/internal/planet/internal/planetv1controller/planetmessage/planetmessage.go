package planetmessage

import (
	"time"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rallies"
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
				Point:          &planetv1.GlobePoint{X: blast.Point.X, Y: blast.Point.Y, Z: blast.Point.Z},
				Version:        Version(version),
				Struck:         struck,
			},
		},
	}
}

func ProvinceFortified(fortification *clicks.Fortification, version clicks.MapVersion) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_ProvinceFortified{
			ProvinceFortified: &planetv1.ProvinceFortified{
				ProvinceId: uint32(fortification.Province),
				CountryId:  fortification.Country,
				TileId:     fortification.Tile,
				Version:    Version(version),
			},
		},
	}
}

func KnockOut(knockOut *clicks.KnockOut, version clicks.MapVersion) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_KnockOut{
			KnockOut: &planetv1.KnockOut{
				CountryId:      knockOut.Country,
				ByCountryId:    knockOut.By,
				TileId:         knockOut.Tile,
				ClearedTileIds: knockOut.Cleared,
				Version:        Version(version),
			},
		},
	}
}

func CutOff(cutOff *clicks.CutOff, version clicks.MapVersion) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_CutOff{
			CutOff: &planetv1.CutOff{
				CountryId:      cutOff.Country,
				ByCountryId:    cutOff.By,
				TileId:         cutOff.Tile,
				ClearedTileIds: cutOff.Cleared,
				Version:        Version(version),
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
				Provinces:  ProvinceHoldings(tally.Provinces),
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

func ProvinceHoldings(rows []clicks.ProvinceTiles) []*planetv1.ProvinceHoldings {
	provinces := make([]*planetv1.ProvinceHoldings, len(rows))
	for i, row := range rows {
		provinces[i] = &planetv1.ProvinceHoldings{ProvinceId: uint32(row.Province), Holdings: Holdings(row.Flags)}
	}
	return provinces
}

func TilesSpread(country string, tile uint32, spread []uint32) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_TilesSpread{
			TilesSpread: &planetv1.TilesSpread{CountryId: country, TileId: tile, SpreadTileIds: spread},
		},
	}
}

func TilesEnclosed(enclosure *clicks.Enclosure, version clicks.MapVersion, yours bool) *planetv1.PlanetEvent {
	struck := make([]*planetv1.TileShields, len(enclosure.Struck))
	for i, tile := range enclosure.Struck {
		struck[i] = &planetv1.TileShields{TileId: tile.Tile, Shields: uint32(tile.Shields)} //nolint:gosec // a byte in the tile storage.
	}

	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_TilesEnclosed{
			TilesEnclosed: &planetv1.TilesEnclosed{
				CountryId:     enclosure.Country,
				ClosingTileId: enclosure.ClosingTile,
				WallTileIds:   enclosure.Wall,
				FilledTileIds: enclosure.Filled,
				Yours:         yours,
				Struck:        struck,
				Version:       Version(version),
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
		KnockedOut:  batch.KnockedOut,
		Infected:    infectedOf(batch.Infected),
		Seats:       seatsOf(batch.Seats),
	}
}

func seatsOf(capitals []clicks.Capital) []*planetv1.Seat {
	seats := make([]*planetv1.Seat, len(capitals))
	for i, seat := range capitals {
		seats[i] = &planetv1.Seat{Flag: seat.Country, TileId: seat.Tile}
	}
	return seats
}

func infectedOf(tiles []clicks.InfectedTile) []*planetv1.InfectedTile {
	infected := make([]*planetv1.InfectedTile, len(tiles))
	for i, tile := range tiles {
		infected[i] = &planetv1.InfectedTile{TileId: tile.Tile, VirusId: string(tile.Virus)}
	}
	return infected
}

func VirusSpread(infection *clicks.Infection, version clicks.MapVersion) *planetv1.PlanetEvent {
	struck := make([]*planetv1.TileShields, len(infection.Struck))
	for i, tile := range infection.Struck {
		struck[i] = &planetv1.TileShields{TileId: tile.Tile, Shields: uint32(tile.Shields)} //nolint:gosec // a byte in the tile storage.
	}

	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_VirusSpread{
			VirusSpread: &planetv1.VirusSpread{
				VirusId:         string(infection.Virus.ID),
				CountryId:       infection.Virus.Flag,
				VictimCountryId: infection.Virus.Victim,
				TakenTileIds:    infection.Taken,
				Struck:          struck,
				Ended:           infection.Virus.Ended,
				Version:         Version(version),
			},
		},
	}
}

func MinesExploded(explosion *clicks.Explosion, version clicks.MapVersion) *planetv1.PlanetEvent {
	blasts := make([]*planetv1.MineBlast, len(explosion.Blasts))
	for i, blast := range explosion.Blasts {
		struck := make([]*planetv1.TileShields, len(blast.Struck))
		for j, tile := range blast.Struck {
			struck[j] = &planetv1.TileShields{TileId: tile.Tile, Shields: uint32(tile.Shields)} //nolint:gosec // a byte in the tile storage.
		}
		blasts[i] = &planetv1.MineBlast{
			TileId: blast.Mine.Tile, CountryId: blast.Mine.Flag, TakenTileIds: blast.Taken, Struck: struck,
		}
	}

	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_MinesExploded{
			MinesExploded: &planetv1.MinesExploded{Blasts: blasts, Version: Version(version)},
		},
	}
}

func TrapLaid(flag string, tile uint32, version clicks.MapVersion) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_TrapLaid{
			TrapLaid: &planetv1.TrapLaid{CountryId: flag, TileId: tile, Version: Version(version)},
		},
	}
}

func TrapSprung(tile uint32, flag string, clicks float64, yours bool) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_TrapSprung{
			TrapSprung: &planetv1.TrapSprung{TileId: tile, CountryId: flag, Clicks: clicks, Yours: yours},
		},
	}
}

func MineLaid(flag string, tile uint32, version clicks.MapVersion) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_MineLaid{
			MineLaid: &planetv1.MineLaid{CountryId: flag, TileId: tile, Version: Version(version)},
		},
	}
}

// Rally is a rally as every flag sees it, with the time it has left as it is told.
func Rally(rally rallies.Snapshot, left time.Duration) *planetv1.Rally {
	return &planetv1.Rally{
		RallyId:   rally.ID.String(),
		AccountId: rally.Account.String(),
		CountryId: rally.Flag,
		Point:     &planetv1.GlobePoint{X: rally.Centre.X, Y: rally.Centre.Y, Z: rally.Centre.Z},
		Radius:    rally.Radius,
		LeftMs:    uint32(max(left, 0).Milliseconds()), //nolint:gosec // a rally lasts minutes.
	}
}

func RallyCalled(rally rallies.Snapshot, left time.Duration) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{Event: &planetv1.PlanetEvent_RallyCalled{RallyCalled: Rally(rally, left)}}
}

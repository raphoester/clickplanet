package paint

import (
	"cmp"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
)

// One tile a player took for a flag, as planet's ledger keeps it.
type Take struct {
	Account cpsession.AccountID
	Tile    drawings.TileID
	Flag    string
	At      time.Time
}

// What one player painted with one flag: each tile once, smallest first.
type Painting struct {
	Account cpsession.AccountID
	Flag    string
	Tiles   []drawings.TileID
}

// Every painting of least tiles or more in the takes, which come oldest first; by account, then flag. A player's latest
// take of a tile is the one that counts: a tile it took again for another flag is painted with that one.
func PaintingsOf(takes []Take, least int) []Painting {
	latest := map[cpsession.AccountID]map[drawings.TileID]string{}
	for _, take := range takes {
		if take.Account == cpsession.NoAccount {
			continue
		}
		tiles, ok := latest[take.Account]
		if !ok {
			tiles = map[drawings.TileID]string{}
			latest[take.Account] = tiles
		}
		tiles[take.Tile] = take.Flag
	}

	var paintings []Painting
	for account, tiles := range latest {
		byFlag := map[string][]drawings.TileID{}
		for tile, flag := range tiles {
			byFlag[flag] = append(byFlag[flag], tile)
		}
		for flag, painted := range byFlag {
			if len(painted) >= least {
				slices.Sort(painted)
				paintings = append(paintings, Painting{Account: account, Flag: flag, Tiles: painted})
			}
		}
	}
	slices.SortFunc(paintings, func(a, b Painting) int {
		if c := cmp.Compare(a.Account.String(), b.Account.String()); c != 0 {
			return c
		}
		return cmp.Compare(a.Flag, b.Flag)
	})
	return paintings
}

// The tiles of the painting the player still holds for its flag on the map: a tile taken since, cleared by a bomb or
// reverted is no longer its drawing.
func (p Painting) Held(onMap map[drawings.TileID]drawings.MapTile) []drawings.Tile {
	var held []drawings.Tile
	for _, id := range p.Tiles {
		if tile, ok := onMap[id]; ok && tile.Holder == p.Account && tile.Flag == p.Flag {
			held = append(held, tile.Tile)
		}
	}
	return held
}

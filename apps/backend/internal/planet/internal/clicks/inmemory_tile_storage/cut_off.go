package inmemory_tile_storage

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// cutOffLocked clears the land of the flag its losses cut from its capital, over the links of the board in play: every
// tile of it goes blank, with no shield and no holder.
func (s *Storage) cutOffLocked(losses clicks.Losses) (clicks.CutOff, error) {
	code, known := s.board.codeOf(losses.Flag)
	holds := func(tile uint32) bool {
		return known && tile <= s.board.last() && s.board.tiles[tile].owner == code
	}
	cut, err := s.capitals.Stranded(losses.Flag, losses.Tiles, s.played, holds)
	if err != nil {
		return clicks.CutOff{}, err //nolint:wrapcheck // the sentinel, unwrapped: every write that takes asks.
	}

	for _, tile := range cut {
		s.moveLocked(tile, unownedCode)
	}

	return clicks.CutOff{Country: losses.Flag, By: losses.By, Tile: losses.Tiles[len(losses.Tiles)-1], Cleared: cut}, nil
}

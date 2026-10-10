package inmemory_tile_storage

import (
	"context"
	"fmt"
	"math"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func (s *Storage) Fortify(_ context.Context, tile uint32, flag string, most int) (clicks.Fortification, error) {
	landmass := s.borders.LandmassOf(tile)

	s.tilesMu.RLock()
	err := s.fortifyErrorLocked(landmass, flag)
	s.tilesMu.RUnlock()
	if err != nil {
		return clicks.Fortification{}, err
	}

	s.tilesMu.Lock()
	if err := s.fortifyErrorLocked(landmass, flag); err != nil {
		s.tilesMu.Unlock()
		return clicks.Fortification{}, err
	}

	most = min(most, math.MaxUint8)
	members := s.borders.TilesOf(landmass)
	raised := make([]clicks.TileShields, 0, len(members))
	for _, member := range members {
		if int(s.tiles[member].shields) < most {
			s.tiles[member].shields++
			s.markDirtyLocked(member)
			raised = append(raised, clicks.TileShields{Tile: member, Shields: int(s.tiles[member].shields)})
		}
	}
	s.fortifiedBy[landmass] = s.codeIDs[flag]
	s.markLandmassDirtyLocked(landmass)
	s.tilesMu.Unlock()

	fortification := clicks.Fortification{
		Landmass: landmass, Ground: s.borders.CountryOf(tile), Country: flag, Tile: tile, Tiles: len(members), Raised: raised,
	}
	s.publish(clicks.Change{Fortification: &fortification})

	return fortification, nil
}

// The flag each locked landmass is locked to: the one that fortified it last, or settled on it whole.
func (s *Storage) Fortresses() map[clicks.LandmassID]string {
	s.tilesMu.RLock()
	defer s.tilesMu.RUnlock()

	fortresses := map[clicks.LandmassID]string{}
	for landmass, code := range s.fortifiedBy {
		if code != unownedCode {
			fortresses[clicks.LandmassID(landmass)] = s.codes[code] //nolint:gosec // landmasses are uint16 in the blob.
		}
	}

	return fortresses
}

func (s *Storage) fortifyErrorLocked(landmass clicks.LandmassID, flag string) error {
	held := 0
	if id, ok := s.codeIDs[flag]; ok {
		held = s.heldLocked(landmass, id)
	}

	if err := clicks.FortifyError(
		landmass, flag, held, len(s.borders.TilesOf(landmass)), s.codes[s.fortifiedBy[landmass]]); err != nil {
		return fmt.Errorf("landmass %d: %w", landmass, err)
	}

	return nil
}

func (s *Storage) moveLocked(tile uint32, to uint16) {
	from := s.tiles[tile].owner
	landmass := s.borders.LandmassOf(tile)

	if from != unownedCode {
		s.counts[from]--
		*s.landmassCountLocked(landmass, from)--
	}
	if to != unownedCode {
		s.counts[to]++
		*s.landmassCountLocked(landmass, to)++
	}

	s.tiles[tile] = ownedBy(to)
	s.markDirtyLocked(tile)
}

func (s *Storage) holdLocked(tile uint32, state tileState) {
	s.tiles[tile] = state
	if state.owner != unownedCode {
		s.counts[state.owner]++
		*s.landmassCountLocked(s.borders.LandmassOf(tile), state.owner)++
	}
}

func (s *Storage) heldLocked(landmass clicks.LandmassID, code uint16) int {
	held := s.landmassHeld[landmass]
	if int(code) >= len(held) {
		return 0
	}

	return int(held[code])
}

func (s *Storage) landmassCountLocked(landmass clicks.LandmassID, code uint16) *uint32 {
	held := s.landmassHeld[landmass]
	if int(code) >= len(held) {
		held = append(held, make([]uint32, int(code)+1-len(held))...)
		s.landmassHeld[landmass] = held
	}

	return &held[code]
}

// A whole landmass nobody fortified is locked to its holder, so a write that made it whole earns nothing later.
func (s *Storage) settleLocked(updates []clicks.TileUpdate) {
	for _, update := range updates {
		s.settleLandmassLocked(s.borders.LandmassOf(update.Tile))
	}
}

func (s *Storage) settleLandmassLocked(landmass clicks.LandmassID) {
	members := s.borders.TilesOf(landmass)
	if landmass == clicks.NoLandmass || len(members) == 0 {
		return
	}

	holder := s.tiles[members[0]].owner
	settler, ok := clicks.SettlerOf(
		s.codes[holder], s.heldLocked(landmass, holder), len(members), s.codes[s.fortifiedBy[landmass]])
	if !ok {
		return
	}

	s.fortifiedBy[landmass] = s.codeIDs[settler]
	s.markLandmassDirtyLocked(landmass)
}

func (s *Storage) markLandmassDirtyLocked(landmass clicks.LandmassID) {
	s.dirtyLandmasses[landmass/64] |= 1 << (landmass % 64)
}

package inmemory_tile_storage

import (
	"context"
	"fmt"
	"math"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func (s *Storage) Fortify(_ context.Context, tile uint32, flag string, most int) (clicks.Fortification, error) {
	landmass := s.landmasses.of(tile)

	s.mu.RLock()
	err := s.fortifyErrorLocked(landmass, flag)
	s.mu.RUnlock()
	if err != nil {
		return clicks.Fortification{}, err
	}

	s.mu.Lock()
	if err := s.fortifyErrorLocked(landmass, flag); err != nil {
		s.mu.Unlock()
		return clicks.Fortification{}, err
	}

	most = min(most, math.MaxUint8)
	members := s.landmasses.tilesOf(landmass)
	raised := make([]clicks.TileShields, 0, len(members))
	for _, member := range members {
		if s.board.shieldsOn(member) < most {
			raised = append(raised, clicks.TileShields{Tile: member, Shields: s.board.raise(member)})
		}
	}
	s.lockLocked(landmass, flag)
	s.mu.Unlock()

	fortification := clicks.Fortification{
		Landmass: landmass, Ground: s.landmasses.groundOf(tile), Country: flag, Tile: tile,
		Tiles: len(members), Raised: raised,
	}
	s.feed.publish(clicks.Change{Fortification: &fortification})

	return fortification, nil
}

// The flag each locked landmass is locked to: the one that fortified it last, or settled on it whole.
func (s *Storage) Fortresses() map[clicks.LandmassID]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fortresses := map[clicks.LandmassID]string{}
	s.landmasses.each(func(landmass clicks.LandmassID) {
		if flag := s.lockedToLocked(landmass); flag != "" {
			fortresses[landmass] = flag
		}
	})

	return fortresses
}

func (s *Storage) fortifyErrorLocked(landmass clicks.LandmassID, flag string) error {
	if err := clicks.FortifyError(
		landmass, flag, s.heldLocked(landmass, flag), len(s.landmasses.tilesOf(landmass)), s.lockedToLocked(landmass),
	); err != nil {
		return fmt.Errorf("landmass %d: %w", landmass, err)
	}

	return nil
}

// A whole landmass nobody fortified is locked to its holder, so a write that made it whole earns nothing later.
func (s *Storage) settleLocked(updates []clicks.TileUpdate) {
	for _, update := range updates {
		s.settleLandmassLocked(s.landmasses.of(update.Tile))
	}
}

func (s *Storage) settleLandmassLocked(landmass clicks.LandmassID) {
	members := s.landmasses.tilesOf(landmass)
	if landmass == clicks.NoLandmass || len(members) == 0 {
		return
	}

	holder := s.board.ownerOf(members[0])
	settler, ok := clicks.SettlerOf(holder, s.heldLocked(landmass, holder), len(members), s.lockedToLocked(landmass))
	if ok {
		s.lockLocked(landmass, settler)
	}
}

func (s *Storage) heldLocked(landmass clicks.LandmassID, flag string) int {
	id, ok := s.board.codeOf(flag)
	if !ok {
		return 0
	}
	return s.landmasses.heldBy(landmass, id)
}

func (s *Storage) lockedToLocked(landmass clicks.LandmassID) string {
	return s.board.nameOf(s.landmasses.lockedTo(landmass))
}

// The flag holds tiles of the landmass, so the board already knows its code.
func (s *Storage) lockLocked(landmass clicks.LandmassID, flag string) {
	id, _ := s.board.codeOf(flag)
	s.landmasses.lock(landmass, id)
}

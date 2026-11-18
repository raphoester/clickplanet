package inmemory_tile_storage

import (
	"context"
	"fmt"
	"math"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

func (s *Storage) fortifyLocked(ctx context.Context, tile uint32, flag string, most int) (clicks.Fortification, error) {
	province := s.provinces.of(tile)
	if err := s.fortifyErrorLocked(province, flag); err != nil {
		return clicks.Fortification{}, err
	}

	most = min(most, math.MaxUint8)
	members := s.provinces.tilesOf(province)
	raised := make([]clicks.TileShields, 0, len(members))
	for _, member := range members {
		if s.board.shieldsOn(member) < most && !s.capitalLocked(member) {
			raised = append(raised, clicks.TileShields{Tile: member, Shields: s.board.raise(member)})
		}
	}
	s.lockLocked(province, flag)

	return clicks.Fortification{
		Province: province, Asset: s.provinces.asset(), Ground: s.landmasses.groundOf(tile), Country: flag, Tile: tile,
		Account: cpctx.GetAccount(ctx), Tiles: len(members), Raised: raised,
	}, nil
}

// The flag each locked province is locked to: the one that fortified it last, or settled on it whole.
func (s *Storage) Fortresses() map[clicks.ProvinceID]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fortresses := map[clicks.ProvinceID]string{}
	s.provinces.each(func(province clicks.ProvinceID) {
		if flag := s.lockedToLocked(province); flag != "" {
			fortresses[province] = flag
		}
	})

	return fortresses
}

func (s *Storage) fortifyErrorLocked(province clicks.ProvinceID, flag string) error {
	if err := clicks.FortifyError(
		province, flag, s.heldLocked(province, flag), len(s.provinces.tilesOf(province)), s.lockedToLocked(province),
	); err != nil {
		return fmt.Errorf("province %d: %w", province, err)
	}

	return nil
}

// A whole province nobody fortified is locked to its holder, so a write that made it whole earns nothing later.
func (s *Storage) settleLocked(updates []clicks.TileUpdate) {
	for _, update := range updates {
		s.settleProvinceLocked(s.provinces.of(update.Tile))
	}
}

func (s *Storage) settleProvinceLocked(province clicks.ProvinceID) {
	members := s.provinces.tilesOf(province)
	if province == clicks.NoProvince || len(members) == 0 {
		return
	}

	holder := s.board.ownerOf(members[0])
	settler, ok := clicks.SettlerOf(holder, s.heldLocked(province, holder), len(members), s.lockedToLocked(province))
	if ok {
		s.lockLocked(province, settler)
	}
}

func (s *Storage) heldLocked(province clicks.ProvinceID, flag string) int {
	id, ok := s.board.codeOf(flag)
	if !ok {
		return 0
	}
	return s.provinces.heldBy(province, id)
}

func (s *Storage) lockedToLocked(province clicks.ProvinceID) string {
	return s.board.nameOf(s.provinces.lockedTo(province))
}

// The flag holds tiles of the province, so the board already knows its code.
func (s *Storage) lockLocked(province clicks.ProvinceID, flag string) {
	id, _ := s.board.codeOf(flag)
	s.provinces.lock(province, id)
}

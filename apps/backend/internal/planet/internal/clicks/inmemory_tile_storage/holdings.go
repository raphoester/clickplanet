package inmemory_tile_storage

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type holding struct {
	holder uint32
	owner  uint16
}

func (s *Storage) HoldingsOf(landmass clicks.LandmassID) []clicks.Holding {
	s.mu.RLock()
	defer s.mu.RUnlock()

	counts := map[holding]int{}
	for _, tile := range s.landmasses.tilesOf(landmass) {
		state := s.board.tiles[tile]
		if state.owner != unownedCode && state.holder != noHolder {
			counts[holding{holder: state.holder, owner: state.owner}]++
		}
	}

	holdings := make([]clicks.Holding, 0, len(counts))
	for key, tiles := range counts {
		holdings = append(holdings, clicks.Holding{
			Account: s.board.holders.accountOf(key.holder), Flag: s.board.nameOf(key.owner), Tiles: tiles,
		})
	}

	return holdings
}

// The tiles keep their flag and their shields: only who holds them is forgotten.
func (s *Storage) ForgetHolder(_ context.Context, account cpsession.AccountID) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, ok := s.board.holders.idOf(account)
	if !ok {
		return 0, nil
	}

	released := 0
	for tile := uint32(1); tile <= s.board.last(); tile++ {
		if s.board.tiles[tile].holder == id {
			s.board.release(tile)
			released++
		}
	}
	s.board.holders.forget(account)

	return released, nil
}

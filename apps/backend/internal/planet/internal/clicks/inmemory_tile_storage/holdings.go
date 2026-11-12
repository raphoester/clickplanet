package inmemory_tile_storage

import (
	"context"
	"fmt"

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

// Each tile asked, in the order asked, under one read of the lock: a tile off the map is nobody's.
func (s *Storage) HoldsOf(tiles []uint32) []clicks.TileHold {
	s.mu.RLock()
	defer s.mu.RUnlock()

	holds := make([]clicks.TileHold, len(tiles))
	for i, tile := range tiles {
		holds[i] = clicks.TileHold{Tile: tile, Holder: cpsession.NoAccount}
		if tile == 0 || tile > s.board.last() {
			continue
		}
		state := s.board.tiles[tile]
		holds[i].Flag = s.board.nameOf(state.owner)
		holds[i].Holder = s.board.holders.accountOf(state.holder)
	}

	return holds
}

// The tiles keep their flag and their shields: only who holds them is forgotten.
// ForgetHolder takes the account off every tile in memory, then off every tile postgres keeps: the maps that do not play
// too, so a map played again names it no more.
func (s *Storage) ForgetHolder(ctx context.Context, account cpsession.AccountID) (int, error) {
	released := s.forgetHolderInMemory(account)
	if err := s.persistence.ForgetHolder(ctx, account); err != nil {
		return released, fmt.Errorf("failed to forget a holder in every map: %w", err)
	}
	return released, nil
}

func (s *Storage) forgetHolderInMemory(account cpsession.AccountID) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, ok := s.board.holders.idOf(account)
	if !ok {
		return 0
	}

	released := 0
	for tile := uint32(1); tile <= s.board.last(); tile++ {
		if s.board.tiles[tile].holder == id {
			s.board.release(tile)
			released++
		}
	}
	s.board.holders.forget(account)

	return released
}

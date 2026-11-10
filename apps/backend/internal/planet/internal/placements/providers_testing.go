//go:build testing

package placements

import (
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type SequentialIDs struct {
	mu   sync.Mutex
	next byte
}

func (s *SequentialIDs) NewID() (clicks.PlacementID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	return clicks.PlacementID{15: s.next}, nil
}

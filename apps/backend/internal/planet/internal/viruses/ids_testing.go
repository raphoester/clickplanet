//go:build testing

package viruses

import (
	"fmt"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type SequentialIDs struct {
	mu   sync.Mutex
	next int
}

func (s *SequentialIDs) NewID() (clicks.VirusID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	return clicks.VirusID(fmt.Sprintf("00000000-0000-7000-8000-%012d", s.next)), nil
}

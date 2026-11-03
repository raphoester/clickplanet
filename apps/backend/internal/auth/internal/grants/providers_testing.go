//go:build testing

package grants

import "sync"

type SequentialIDs struct {
	mu   sync.Mutex
	next byte
}

func (s *SequentialIDs) NewID() (GrantID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	return GrantID{0: 0x9, 15: s.next}, nil
}

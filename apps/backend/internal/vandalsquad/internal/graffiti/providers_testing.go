//go:build testing

package graffiti

import "sync"

type SequentialIDs struct {
	mu   sync.Mutex
	next byte
}

func (s *SequentialIDs) NewID() (ID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	return ID{15: s.next}, nil
}

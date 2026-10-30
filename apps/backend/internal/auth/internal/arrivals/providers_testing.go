//go:build testing

package arrivals

import "sync"

type SequentialIDs struct {
	mu   sync.Mutex
	next byte
}

func (s *SequentialIDs) NewID() (ArrivalID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	return ArrivalID{0: 0xa, 15: s.next}, nil
}

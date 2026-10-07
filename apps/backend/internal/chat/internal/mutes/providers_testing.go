//go:build testing

package mutes

import "sync"

type SequentialIDs struct {
	mu   sync.Mutex
	next byte
}

func (s *SequentialIDs) NewID() (MuteID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	return MuteID{15: s.next}, nil
}

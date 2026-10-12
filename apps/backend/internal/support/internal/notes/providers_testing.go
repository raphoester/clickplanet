//go:build testing

package notes

import "sync"

type SequentialIDs struct {
	mu   sync.Mutex
	next byte
}

func (s *SequentialIDs) NewID() (NoteID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	return NoteID{15: s.next}, nil
}

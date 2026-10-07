//go:build testing

package announcements

import "sync"

type SequentialIDs struct {
	mu   sync.Mutex
	next byte
}

func (s *SequentialIDs) NewID() (AnnouncementID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	return AnnouncementID{15: s.next}, nil
}

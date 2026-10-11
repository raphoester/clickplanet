//go:build testing

package reports

import "sync"

type SequentialIDs struct {
	mu   sync.Mutex
	next byte
}

func (s *SequentialIDs) NewID() (ReportID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	return ReportID{15: s.next}, nil
}

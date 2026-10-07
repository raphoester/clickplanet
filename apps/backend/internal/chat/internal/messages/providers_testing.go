//go:build testing

package messages

import (
	"sync"

	"github.com/google/uuid"
)

type SequentialIDs struct {
	mu   sync.Mutex
	next byte
}

func (s *SequentialIDs) NewID() (MessageID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	return MessageID(uuid.UUID{15: s.next}.String()), nil
}

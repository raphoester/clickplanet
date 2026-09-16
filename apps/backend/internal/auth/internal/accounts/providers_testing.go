//go:build testing

package accounts

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// SequentialIDs answers uuid.UUID{15: 1}, then {15: 2}, and so on.
type SequentialIDs struct {
	mu   sync.Mutex
	next byte
}

func (s *SequentialIDs) NewID() (uuid.UUID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	return uuid.UUID{15: s.next}, nil
}

// SequentialTokens answers token-1, then token-2, and so on.
type SequentialTokens struct {
	mu   sync.Mutex
	next int
}

func (s *SequentialTokens) NewToken() (*Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	return TokenOf(fmt.Sprintf("token-%d", s.next)), nil
}

//go:build testing

package accounts

import (
	"fmt"
	"sync"
)

type SequentialIDs struct {
	mu   sync.Mutex
	next byte
}

func (s *SequentialIDs) NewID() (AccountID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	return AccountID{15: s.next}, nil
}

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

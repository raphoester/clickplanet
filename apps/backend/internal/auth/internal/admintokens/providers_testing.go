//go:build testing

package admintokens

import (
	"fmt"
	"sync"
)

type SequentialSecrets struct {
	mu   sync.Mutex
	next int
}

func (s *SequentialSecrets) NewSecret() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	return fmt.Sprintf("secret-%d", s.next), nil
}

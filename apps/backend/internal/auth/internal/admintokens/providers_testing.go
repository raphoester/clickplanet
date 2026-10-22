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

func (s *SequentialSecrets) NewSecret(prefix string) (Secret, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	return Secret(fmt.Sprintf("%ssecret-%d", prefix, s.next)), nil
}

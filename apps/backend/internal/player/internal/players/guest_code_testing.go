//go:build testing

package players

import (
	"fmt"
	"sync"
)

// SequentialCodes answers 000001, then 000002, and so on.
type SequentialCodes struct {
	mu   sync.Mutex
	next int
}

func (s *SequentialCodes) NewGuestCode() (GuestCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	return GuestCode(fmt.Sprintf("%06x", s.next)), nil
}

// RepeatedCodes answers the codes given in turn, then the last one forever: a generator that repeats itself.
type RepeatedCodes struct {
	mu    sync.Mutex
	codes []GuestCode
}

func NewRepeatedCodes(codes ...GuestCode) *RepeatedCodes {
	return &RepeatedCodes{codes: codes}
}

func (r *RepeatedCodes) NewGuestCode() (GuestCode, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	code := r.codes[0]
	if len(r.codes) > 1 {
		r.codes = r.codes[1:]
	}
	return code, nil
}

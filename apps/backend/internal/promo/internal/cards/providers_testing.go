//go:build testing

package cards

import (
	"context"
	"sync"
)

type MemoryStore struct {
	mu    sync.Mutex
	decks map[SetName]DeckSnapshot
	err   error
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{decks: map[SetName]DeckSnapshot{}}
}

func (s *MemoryStore) Failing(err error) *MemoryStore {
	s.err = err
	return s
}

func (s *MemoryStore) Save(_ context.Context, deck Deck) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	snapshot := deck.TakeSnapshot()
	s.decks[snapshot.Set] = snapshot
	return nil
}

func (s *MemoryStore) Deck(set SetName) (DeckSnapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	deck, ok := s.decks[set]
	return deck, ok
}

// FakeSets is planet's names.
type FakeSets []string

func (f FakeSets) Names(context.Context) ([]string, error) { return f, nil }

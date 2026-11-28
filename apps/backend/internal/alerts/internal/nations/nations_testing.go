//go:build testing

package nations

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"
)

type InMemoryStore struct {
	mu   sync.Mutex
	sold map[Flag]Sold
}

var _ Store = (*InMemoryStore)(nil)

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{sold: map[Flag]Sold{}}
}

func (s *InMemoryStore) SoldFlags(_ context.Context) ([]Sold, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sold := make([]Sold, 0, len(s.sold))
	for _, flag := range s.sold {
		sold = append(sold, flag)
	}
	slices.SortFunc(sold, func(a, b Sold) int { return strings.Compare(string(a.flag), string(b.flag)) })
	return sold, nil
}

func (s *InMemoryStore) SaveSold(_ context.Context, sold Sold) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sold[sold.flag] = sold
	return nil
}

// A catalog of the flags the game ships with, and of the sold ones named here, told at the zero time.
func NewCatalog(sold map[Flag]string) *Catalog {
	catalog := &Catalog{store: NewInMemoryStore(), sold: map[Flag]Sold{}}
	for flag, name := range sold {
		catalog.sold[flag] = Sold{flag: flag, name: name, namedAt: time.Time{}}
	}
	return catalog
}

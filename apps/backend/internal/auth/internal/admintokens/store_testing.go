//go:build testing

package admintokens

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

// The Store in a slice, for the use cases' tests. Kept is what it holds.
type MemoryStore struct {
	mu   sync.Mutex
	kept []Snapshot
}

var _ Store = (*MemoryStore)(nil)

func (m *MemoryStore) SaveTokens(_ context.Context, tokens []Snapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.kept = append(m.kept, tokens...)
	return nil
}

func (m *MemoryStore) SpendRefresh(_ context.Context, hash accounts.TokenHash, now time.Time) (Holder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, token := range m.kept {
		if token.Kind == KindRefresh && slices.Equal(token.Hash, hash) && now.Before(token.ExpiresAt) {
			m.kept = slices.Delete(m.kept, i, i+1)
			return token.Holder, nil
		}
	}
	return Holder{}, ErrUnknownToken
}

func (m *MemoryStore) DeleteLapsed(_ context.Context, now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.kept = slices.DeleteFunc(m.kept, func(token Snapshot) bool { return !now.Before(token.ExpiresAt) })
	return nil
}

func (m *MemoryStore) Kept() []Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.kept)
}

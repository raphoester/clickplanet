//go:build testing

package admintokens

import (
	"context"
	"slices"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type MemoryStore struct {
	mu   sync.Mutex
	kept []Snapshot
}

func (m *MemoryStore) Connection(_ context.Context, hash accounts.TokenHash) ([]*Token, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	found := slices.IndexFunc(m.kept, func(token Snapshot) bool { return slices.Equal(token.Hash, hash) })
	if found < 0 {
		return nil, nil
	}
	var connection []*Token
	for _, token := range m.kept {
		if token.Connection == m.kept[found].Connection {
			restored, err := token.Restore()
			if err != nil {
				return nil, err
			}
			connection = append(connection, restored)
		}
	}
	return connection, nil
}

func (m *MemoryStore) SaveTokens(ctx context.Context, tokens ...Snapshot) error {
	cpatomic.OnCommit(ctx, func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		for _, token := range tokens {
			found := slices.IndexFunc(m.kept, func(kept Snapshot) bool { return slices.Equal(kept.Hash, token.Hash) })
			if found < 0 {
				m.kept = append(m.kept, token)
			} else {
				m.kept[found] = token
			}
		}
	})
	return nil
}

func (m *MemoryStore) Kept() []Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.kept)
}

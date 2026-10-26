//go:build testing

package rules

import (
	"context"
	"sync"
)

var _ Choices = (*MemoryChoices)(nil)

type MemoryChoices struct {
	mu      sync.Mutex
	kept    *Choice
	failing error
}

func NewMemoryChoices() *MemoryChoices {
	return &MemoryChoices{}
}

func (m *MemoryChoices) Choice(context.Context) (Choice, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return Choice{}, false, m.failing
	}
	if m.kept == nil {
		return Choice{}, false, nil
	}
	return *m.kept, true, nil
}

func (m *MemoryChoices) Save(_ context.Context, choice Choice) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	m.kept = &choice
	return nil
}

func (m *MemoryChoices) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failing = err
}

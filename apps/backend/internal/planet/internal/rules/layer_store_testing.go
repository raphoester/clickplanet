//go:build testing

package rules

import (
	"context"
	"sync"
)

var _ LayerStore = (*MemoryLayerStore)(nil)

type MemoryLayerStore struct {
	mu        sync.Mutex
	baselines []BaselineSnapshot
	override  OverrideSnapshot
	failing   error
}

func NewMemoryLayerStore() *MemoryLayerStore {
	return &MemoryLayerStore{}
}

func (m *MemoryLayerStore) Baseline(_ context.Context, request RequestID) (Baseline, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return Baseline{}, m.failing
	}
	for _, kept := range m.baselines {
		if kept.Request == request {
			return kept.Restore(), nil
		}
	}
	return Baseline{}, nil
}

func (m *MemoryLayerStore) LatestBaseline(context.Context) (Baseline, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return Baseline{}, m.failing
	}
	var latest BaselineSnapshot
	for _, kept := range m.baselines {
		if !kept.AppliedAt.Before(latest.AppliedAt) {
			latest = kept
		}
	}
	return latest.Restore(), nil
}

func (m *MemoryLayerStore) SaveBaseline(_ context.Context, baseline Baseline) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	kept := baseline.TakeSnapshot()
	for _, before := range m.baselines {
		if before.Request == kept.Request {
			return nil
		}
	}
	m.baselines = append(m.baselines, kept)
	return nil
}

func (m *MemoryLayerStore) Override(context.Context) (Override, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return Override{}, m.failing
	}
	return m.override.Restore(), nil
}

func (m *MemoryLayerStore) SaveOverride(_ context.Context, override Override) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	m.override = override.TakeSnapshot()
	return nil
}

func (m *MemoryLayerStore) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failing = err
}

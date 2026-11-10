//go:build testing

package inmemory_tile_storage

import (
	"context"
	"slices"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var _ Placements = (*RecordedPlacements)(nil)

type RecordedPlacements struct {
	mu        sync.Mutex
	confirmed []clicks.PlacementID
	failing   error
}

func (r *RecordedPlacements) Confirm(_ context.Context, placed []clicks.PlacementID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.failing != nil {
		return r.failing
	}
	r.confirmed = append(r.confirmed, placed...)
	return nil
}

func (r *RecordedPlacements) Confirmed() []clicks.PlacementID {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.confirmed)
}

func (r *RecordedPlacements) FailWith(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.failing = err
}

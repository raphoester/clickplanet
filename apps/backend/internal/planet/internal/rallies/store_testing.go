//go:build testing

package rallies

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

var _ Store = (*MemoryStore)(nil)

type MemoryStore struct {
	mu      sync.Mutex
	rows    map[cpsession.AccountID]Snapshot
	failing error
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{rows: map[cpsession.AccountID]Snapshot{}}
}

func (m *MemoryStore) Last(_ context.Context, account cpsession.AccountID) (Rally, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return Rally{}, m.failing
	}
	return m.rows[account].Restore(), nil
}

func (m *MemoryStore) Change(_ context.Context, account cpsession.AccountID, change func(Rally) (Rally, error)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	after, err := change(m.rows[account].Restore())
	if err != nil {
		return err
	}
	m.rows[account] = after.TakeSnapshot()
	return nil
}

func (m *MemoryStore) On(_ context.Context, flag string, at time.Time) ([]Rally, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return nil, m.failing
	}
	var on []Rally
	for _, row := range m.rows {
		if row.Flag == flag && row.EndsAt.After(at) {
			on = append(on, row.Restore())
		}
	}
	return on, nil
}

func (m *MemoryStore) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failing = err
}

type SequentialIDs struct {
	mu   sync.Mutex
	next int
}

func (s *SequentialIDs) NewID() (RallyID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	id, err := uuid.Parse(fmt.Sprintf("00000000-0000-7000-8000-%012d", s.next))
	return RallyID(id), err
}

// RecordedCalls is every rally told, in order.
type RecordedCalls struct {
	mu   sync.Mutex
	told []Snapshot
}

var _ Teller = (*RecordedCalls)(nil)

func (r *RecordedCalls) Called(rally Rally) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.told = append(r.told, rally.TakeSnapshot())
}

func (r *RecordedCalls) Told() []Snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Snapshot(nil), r.told...)
}

// Line is a ground of tiles on a great circle, tile n at n hundredths of a radian from tile 1.
type Line struct{}

func (Line) Position(id uint32) (clicks.Vec3, bool) {
	if id == 0 {
		return clicks.Vec3{}, false
	}
	angle := float64(id-1) / 100
	return clicks.Vec3{X: math.Cos(angle), Y: math.Sin(angle)}, true
}

//go:build testing

package eras

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

// Earth1 is earth-1 as the migration that made boards keeps it.
var Earth1 = Board{ID: clicks.TestBoard, Name: "earth-1", Hash: "3fb28bdf"}

// Earth2 is a board planet keeps beside earth-1 in a test, and no process carries.
var Earth2 = Board{ID: clicks.TestBoard + 1, Name: "earth-2", Hash: "0123abcd"}

// Earth1Carried is earth-1 as this process's files name it.
var Earth1Carried = Carried{Name: Earth1.Name, Hash: Earth1.Hash}

// Epoch is when the world's first era started, as the migration keeps it.
var Epoch = time.Unix(0, 0).UTC()

// MemoryStore keeps what the migration keeps: earth-1, and the world on it since the epoch.
type MemoryStore struct {
	mu      sync.Mutex
	boards  []Board
	eras    []Era
	Failing error
}

var _ Store = (*MemoryStore)(nil)

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{boards: []Board{Earth1}, eras: []Era{{Map: clicks.World, Board: Earth1, StartedAt: Epoch}}}
}

// Keep adds a board's row, as the migration that ships it does.
func (m *MemoryStore) Keep(board Board) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.boards = append(m.boards, board)
}

func (m *MemoryStore) Boards(context.Context) ([]Board, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Failing != nil {
		return nil, m.Failing
	}
	return slices.Clone(m.boards), nil
}

func (m *MemoryStore) Latest(_ context.Context, played clicks.MapID) (Era, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Failing != nil {
		return Era{}, m.Failing
	}
	eras := m.erasLocked(played)
	if len(eras) == 0 {
		return Era{}, fmt.Errorf("%w: %s", ErrNoEra, played)
	}
	return eras[len(eras)-1], nil
}

func (m *MemoryStore) Covering(_ context.Context, played clicks.MapID, since, until time.Time) ([]Era, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Failing != nil {
		return nil, m.Failing
	}
	eras := m.erasLocked(played)
	covering := make([]Era, 0, len(eras))
	for _, era := range eras {
		switch {
		case !era.StartedAt.After(since):
			covering = append(covering[:0], era)
		case era.StartedAt.Before(until):
			covering = append(covering, era)
		}
	}
	return covering, nil
}

func (m *MemoryStore) Start(_ context.Context, played clicks.Stage, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Failing != nil {
		return m.Failing
	}
	board := slices.IndexFunc(m.boards, func(board Board) bool { return board.ID == played.Board })
	if board < 0 {
		return fmt.Errorf("no board %d is kept", played.Board)
	}
	if slices.ContainsFunc(m.eras, func(era Era) bool { return era.Map == played.Map && era.StartedAt.Equal(at) }) {
		return nil
	}
	m.eras = append(m.eras, Era{Map: played.Map, Board: m.boards[board], StartedAt: at.UTC()})
	return nil
}

func (m *MemoryStore) erasLocked(played clicks.MapID) []Era {
	var eras []Era
	for _, era := range m.eras {
		if era.Map == played {
			eras = append(eras, era)
		}
	}
	slices.SortFunc(eras, func(a, b Era) int { return cmp.Compare(a.StartedAt.UnixNano(), b.StartedAt.UnixNano()) })
	return eras
}

// Reseed keeps again what the migration that made boards seeds, once a Purge emptied it: earth-1, and the world on it
// since the epoch. Every tile, lock, mine, virus and event names a board.
func Reseed(ctx context.Context, db cppg.Conn) error {
	if _, err := db.On(ctx).ExecContext(ctx, `
		INSERT INTO boards (id, name, hash) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING
	`, Earth1.ID, string(Earth1.Name), Earth1.Hash); err != nil {
		return fmt.Errorf("failed to keep earth-1 again: %w", err)
	}
	if _, err := db.On(ctx).ExecContext(ctx, `
		INSERT INTO eras (map, board, started_at) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING
	`, string(clicks.World), Earth1.ID, Epoch); err != nil {
		return fmt.Errorf("failed to keep the world's era again: %w", err)
	}
	return nil
}

// CarryingEarth1 is a process that carries earth-1 alone, as planet keeps it.
var CarryingEarth1 = Boards{carried: []Board{Earth1}}

// KeepRow keeps a board's row, as the migration that ships it does.
func KeepRow(ctx context.Context, db cppg.Conn, board Board) error {
	if _, err := db.On(ctx).ExecContext(ctx, `INSERT INTO boards (id, name, hash) VALUES ($1, $2, $3)`,
		board.ID, string(board.Name), board.Hash); err != nil {
		return fmt.Errorf("failed to keep board %s: %w", board.Name, err)
	}
	return nil
}

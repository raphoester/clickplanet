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
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpboards"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

// Earth1 is earth-1 as the migration that made boards keeps it.
var Earth1 = Board{ID: clicks.TestBoard, Name: "earth-1", Hash: "3fb28bdf"}

// Earth2 is earth-2 as the migration that ships it keeps it: the world's next board, which nothing plays yet.
var Earth2 = Board{ID: clicks.TestBoard + 1, Name: "earth-2", Hash: "5a61c10a"}

// Siege1 is siege-1 as the migration that ships it keeps it: Capital Siege's board, which nothing plays yet.
var Siege1 = Board{ID: clicks.TestBoard + 2, Name: "siege-1", Hash: "e219e852"}

// Shipped is every board the migrations keep, by id.
var Shipped = []Board{Earth1, Earth2, Siege1}

// Earth1Carried is earth-1 as this process's files name it.
var Earth1Carried = Carried{Name: Earth1.Name, Hash: Earth1.Hash}

// Earth2Carried is earth-2 as this process's files name it.
var Earth2Carried = Carried{Name: Earth2.Name, Hash: Earth2.Hash}

// Siege1Carried is siege-1 as this process's files name it.
var Siege1Carried = Carried{Name: Siege1.Name, Hash: Siege1.Hash}

// Fixture is the tests' second board (cpboards.Fixture), as a test's planet keeps it: a test binary carries it beside
// the shipped boards, under an id no migration takes.
var Fixture = Board{ID: clicks.TestBoard + 98, Name: cpboards.Fixture, Hash: "cb8330a8"}

// FixtureCarried is the fixture as a test binary's files name it.
var FixtureCarried = Carried{Name: Fixture.Name, Hash: Fixture.Hash}

// Epoch is when the world's first era started, as the migration keeps it.
var Epoch = time.Unix(0, 0).UTC()

// MemoryStore keeps what the migrations keep: the shipped boards, and the world on earth-1 since the epoch.
type MemoryStore struct {
	mu      sync.Mutex
	boards  []Board
	eras    []Era
	Failing error
}

var _ Store = (*MemoryStore)(nil)

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{boards: slices.Clone(Shipped), eras: []Era{{Map: clicks.World, Board: Earth1, StartedAt: Epoch}}}
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

// Reseed keeps again what the migrations seed, once a Purge emptied it: the shipped boards, and the world on earth-1
// since the epoch. Every tile, lock, mine, virus and event names a board.
func Reseed(ctx context.Context, db cppg.Conn) error {
	for _, board := range Shipped {
		if _, err := db.On(ctx).ExecContext(ctx, `
			INSERT INTO boards (id, name, hash) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING
		`, board.ID, string(board.Name), board.Hash); err != nil {
			return fmt.Errorf("failed to keep %s again: %w", board.Name, err)
		}
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

// CarryingFixture is a process that carries earth-1 and the fixture, as a test's planet keeps them.
var CarryingFixture = Boards{carried: []Board{Earth1, Fixture}}

// KeepFixture keeps the fixture's row, which a test binary's planet needs since it carries the fixture: no migration
// ships it.
func KeepFixture(ctx context.Context, db cppg.Conn) error {
	if _, err := db.On(ctx).ExecContext(ctx, `INSERT INTO boards (id, name, hash) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		Fixture.ID, string(Fixture.Name), Fixture.Hash); err != nil {
		return fmt.Errorf("failed to keep the fixture: %w", err)
	}
	return nil
}

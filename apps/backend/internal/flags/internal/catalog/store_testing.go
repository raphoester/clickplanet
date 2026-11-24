//go:build testing

package catalog

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"
)

var _ Store = (*MemoryStore)(nil)

type MemoryStore struct {
	mu   sync.Mutex
	rows map[FlagID]Snapshot
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{rows: map[FlagID]Snapshot{}}
}

func (m *MemoryStore) Flag(_ context.Context, id FlagID) (Flag, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.rows[id]
	if !ok {
		return Flag{}, fmt.Errorf("%s: %w", id, ErrUnknownFlag)
	}
	return row.Restore()
}

func (m *MemoryStore) Flags(context.Context) ([]Flag, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rows := make([]Snapshot, 0, len(m.rows))
	for _, row := range m.rows {
		rows = append(rows, row)
	}
	slices.SortFunc(rows, func(a, b Snapshot) int { return strings.Compare(string(a.ID), string(b.ID)) })
	flags := make([]Flag, 0, len(rows))
	for _, row := range rows {
		flag, err := row.Restore()
		if err != nil {
			return nil, err
		}
		flags = append(flags, flag)
	}
	return flags, nil
}

func (m *MemoryStore) Ending(ctx context.Context, at time.Time) ([]Flag, error) {
	flags, err := m.Flags(ctx)
	if err != nil {
		return nil, err
	}
	var ending []Flag
	for _, flag := range flags {
		if kept := flag.TakeSnapshot(); kept.State == Open && !kept.EndsAt.After(at) {
			ending = append(ending, flag)
		}
	}
	return ending, nil
}

func (m *MemoryStore) Save(_ context.Context, flags ...Flag) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, flag := range flags {
		row := flag.TakeSnapshot()
		for id, kept := range m.rows {
			if id != row.ID && strings.EqualFold(kept.Name.String(), row.Name.String()) {
				return fmt.Errorf("%s: %w", row.Name, ErrNameTaken)
			}
		}
		m.rows[row.ID] = row
	}
	return nil
}

// Sequence draws r-0000000001, r-0000000002 and so on.
type Sequence struct {
	mu   sync.Mutex
	next int
}

func (s *Sequence) NewID() (FlagID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	return ParseFlagID(fmt.Sprintf("%s%0*d", FlagIDPrefix, FlagIDLength, s.next))
}

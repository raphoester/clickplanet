//go:build testing

package inmemory_mission_store

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
)

type briefKey struct {
	day  string
	flag string
}

type sheetKey struct {
	day     string
	account missions.AccountID
}

type Store struct {
	mu       sync.Mutex
	briefs   map[briefKey][]missions.MissionSnapshot
	sheets   map[sheetKey]missions.SheetSnapshot
	failWith error
}

var _ missions.Store = (*Store)(nil)

func New() *Store {
	return &Store{briefs: map[briefKey][]missions.MissionSnapshot{}, sheets: map[sheetKey]missions.SheetSnapshot{}}
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failWith = err
}

func (s *Store) Brief(_ context.Context, day missions.Day, flag string) (missions.Brief, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return missions.Brief{}, s.failWith
	}
	kept, ok := s.briefs[briefKey{day: day.String(), flag: flag}]
	if !ok {
		return missions.Brief{}, missions.ErrNoBrief
	}
	return missions.BriefOf(kept) //nolint:wrapcheck // the domain named it.
}

func (s *Store) KeepBrief(_ context.Context, day missions.Day, flag string, brief missions.Brief) (missions.Brief, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return missions.Brief{}, s.failWith
	}
	key := briefKey{day: day.String(), flag: flag}
	if _, ok := s.briefs[key]; !ok {
		s.briefs[key] = brief.Snapshot()
	}
	return missions.BriefOf(s.briefs[key]) //nolint:wrapcheck // the domain named it.
}

func (s *Store) Sheet(_ context.Context, day missions.Day, account missions.AccountID) (missions.Sheet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return missions.Sheet{}, s.failWith
	}
	return missions.SheetOf(s.sheets[sheetKey{day: day.String(), account: account}]), nil
}

func (s *Store) ChangeSheet(
	_ context.Context, day missions.Day, account missions.AccountID, change func(missions.Sheet) missions.Sheet,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	key := sheetKey{day: day.String(), account: account}
	collected := s.sheets[key].Collected
	next := change(missions.SheetOf(s.sheets[key])).Snapshot()
	next.Collected = collected
	s.sheets[key] = next
	return nil
}

func (s *Store) Collect(
	ctx context.Context, day missions.Day, account missions.AccountID, slot missions.Slot, _ time.Time,
	pay func(context.Context) error,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	key := sheetKey{day: day.String(), account: account}
	sheet := missions.SheetOf(s.sheets[key]).Snapshot()
	if sheet.Collected.Contains(slot) {
		return missions.ErrCollected
	}
	if err := pay(ctx); err != nil {
		return err
	}
	sheet.Collected.Add(slot)
	s.sheets[key] = sheet
	return nil
}

func (s *Store) DeleteAccount(_ context.Context, account missions.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	for key := range s.sheets {
		if key.account == account {
			delete(s.sheets, key)
		}
	}
	return nil
}

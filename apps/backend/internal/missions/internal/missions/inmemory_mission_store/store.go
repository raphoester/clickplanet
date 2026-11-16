//go:build testing

package inmemory_mission_store

import (
	"bytes"
	"context"
	"slices"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
)

type dayKey struct {
	day     string
	account missions.AccountID
}

type Store struct {
	mu       sync.Mutex
	briefs   map[dayKey][]missions.MissionSnapshot
	sheets   map[dayKey]missions.SheetSnapshot
	failWith error
}

var _ missions.Store = (*Store)(nil)

func New() *Store {
	return &Store{briefs: map[dayKey][]missions.MissionSnapshot{}, sheets: map[dayKey]missions.SheetSnapshot{}}
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failWith = err
}

func (s *Store) Brief(_ context.Context, day days.Day, account missions.AccountID) (missions.Brief, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return missions.Brief{}, s.failWith
	}
	kept, ok := s.briefs[dayKey{day: day.String(), account: account}]
	if !ok {
		return missions.Brief{}, missions.ErrNoBrief
	}
	return missions.BriefOf(kept) //nolint:wrapcheck // the domain named it.
}

func (s *Store) KeepBrief(
	_ context.Context, day days.Day, account missions.AccountID, brief missions.Brief,
) (missions.Brief, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return missions.Brief{}, s.failWith
	}
	key := dayKey{day: day.String(), account: account}
	if _, ok := s.briefs[key]; !ok {
		s.briefs[key] = brief.Snapshot()
	}
	return missions.BriefOf(s.briefs[key]) //nolint:wrapcheck // the domain named it.
}

func (s *Store) Sheet(_ context.Context, day days.Day, account missions.AccountID) (missions.Sheet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return missions.Sheet{}, s.failWith
	}
	return missions.SheetOf(s.sheets[dayKey{day: day.String(), account: account}]), nil
}

func (s *Store) ChangeSheet(
	_ context.Context, day days.Day, account missions.AccountID, change func(missions.Sheet) missions.Sheet,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	key := dayKey{day: day.String(), account: account}
	s.sheets[key] = change(missions.SheetOf(s.sheets[key])).Snapshot()
	return nil
}

func (s *Store) PaidInAll(_ context.Context, account missions.AccountID) (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return 0, s.failWith
	}
	var paid uint64
	for key, sheet := range s.sheets {
		if key.account == account && sheet.Paid != nil {
			paid += uint64(sheet.Paid.Len())
		}
	}
	return paid, nil
}

func (s *Store) DoneBy(_ context.Context, account missions.AccountID) (missions.Done, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return missions.Done{}, s.failWith
	}
	return s.doneBy(account)
}

func (s *Store) DoneAfter(_ context.Context, after missions.AccountID, limit int) ([]missions.AccountDone, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	var accounts []missions.AccountID
	for key := range s.sheets {
		if bytes.Compare(key.account[:], after[:]) > 0 && !slices.Contains(accounts, key.account) {
			accounts = append(accounts, key.account)
		}
	}
	slices.SortFunc(accounts, func(a, b missions.AccountID) int { return bytes.Compare(a[:], b[:]) })

	page := make([]missions.AccountDone, 0, limit)
	for _, account := range accounts {
		if len(page) == limit {
			break
		}
		done, err := s.doneBy(account)
		if err != nil {
			return nil, err
		}
		if done.InAll > 0 {
			page = append(page, missions.AccountDone{Account: account, Done: done})
		}
	}
	return page, nil
}

func (s *Store) doneBy(account missions.AccountID) (missions.Done, error) {
	done := missions.NothingDone()
	for key, sheet := range s.sheets {
		if key.account != account {
			continue
		}
		brief, err := missions.BriefOf(s.briefs[key])
		if err != nil {
			return missions.Done{}, err //nolint:wrapcheck // the domain named it.
		}
		done = done.WithDay(brief, missions.SheetOf(sheet))
	}
	return done, nil
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
	for key := range s.briefs {
		if key.account == account {
			delete(s.briefs, key)
		}
	}
	return nil
}

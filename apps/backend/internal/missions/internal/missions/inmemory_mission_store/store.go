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

func (s *Store) PaidTotals(_ context.Context, account missions.AccountID) (missions.Paid, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return missions.Paid{}, s.failWith
	}
	return s.paidTo(account)
}

func (s *Store) PaidAfter(_ context.Context, after missions.AccountID, limit int) ([]missions.AccountPaid, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	var accounts []missions.AccountID
	for key, sheet := range s.sheets {
		if !sheet.Paid.Empty() && bytes.Compare(key.account[:], after[:]) > 0 && !slices.Contains(accounts, key.account) {
			accounts = append(accounts, key.account)
		}
	}
	slices.SortFunc(accounts, func(a, b missions.AccountID) int { return bytes.Compare(a[:], b[:]) })

	page := make([]missions.AccountPaid, 0, min(limit, len(accounts)))
	for _, account := range accounts[:min(limit, len(accounts))] {
		totals, err := s.paidTo(account)
		if err != nil {
			return nil, err
		}
		page = append(page, missions.AccountPaid{Account: account, Paid: totals})
	}
	return page, nil
}

func (s *Store) paidTo(account missions.AccountID) (missions.Paid, error) {
	paid := missions.NothingPaid()
	for key, sheet := range s.sheets {
		if key.account != account {
			continue
		}
		brief, err := missions.BriefOf(s.briefs[key])
		if err != nil {
			return missions.Paid{}, err //nolint:wrapcheck // the domain named it.
		}
		paid = paid.WithDay(brief, missions.SheetOf(sheet))
	}
	return paid, nil
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

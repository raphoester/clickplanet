//go:build testing

package inmemory_report_storage

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
)

func New() *Storage {
	return &Storage{}
}

type Storage struct {
	mu   sync.Mutex
	kept []reports.Snapshot
	err  error
}

var _ reports.Storage = (*Storage)(nil)

func (s *Storage) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.err = err
}

func (s *Storage) Seed(snapshots ...reports.Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.kept = append(s.kept, snapshots...)
}

func (s *Storage) Kept() []reports.Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.oldestFirst(func(reports.Snapshot) bool { return true })
}

func (s *Storage) Save(_ context.Context, saved ...reports.Report) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return s.err
	}
	for _, report := range saved {
		snapshot := report.TakeSnapshot()
		s.kept = slices.DeleteFunc(s.kept, func(kept reports.Snapshot) bool { return kept.ID == snapshot.ID })
		if !snapshot.Forgotten {
			s.kept = append(s.kept, snapshot)
		}
	}
	return nil
}

func (s *Storage) FiledBy(_ context.Context, reporter reports.AccountID, about reports.AccountID) ([]reports.Report, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return nil, s.err
	}
	return restored(s.oldestFirst(func(kept reports.Snapshot) bool {
		return kept.Reporter == reporter && kept.Reported == about
	}))
}

func (s *Storage) Naming(_ context.Context, account reports.AccountID) ([]reports.Report, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return nil, s.err
	}
	return restored(s.oldestFirst(func(kept reports.Snapshot) bool {
		return kept.Reporter == account || kept.Reported == account
	}))
}

func (s *Storage) FiledBefore(_ context.Context, cutoff time.Time, limit int) ([]reports.Report, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return nil, s.err
	}
	old := s.oldestFirst(func(kept reports.Snapshot) bool { return kept.At.Before(cutoff) })
	return restored(old[:min(len(old), limit)])
}

func (s *Storage) oldestFirst(wanted func(reports.Snapshot) bool) []reports.Snapshot {
	var found []reports.Snapshot
	for _, kept := range s.kept {
		if wanted(kept) {
			found = append(found, kept)
		}
	}
	slices.SortStableFunc(found, func(a, b reports.Snapshot) int { return a.At.Compare(b.At) })
	return found
}

func restored(snapshots []reports.Snapshot) ([]reports.Report, error) {
	found := make([]reports.Report, 0, len(snapshots))
	for _, snapshot := range snapshots {
		report, err := snapshot.Restore()
		if err != nil {
			return nil, fmt.Errorf("failed to restore a report: %w", err)
		}
		found = append(found, report)
	}
	return found, nil
}

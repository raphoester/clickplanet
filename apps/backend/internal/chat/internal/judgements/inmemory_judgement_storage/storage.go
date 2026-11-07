//go:build testing

package inmemory_judgement_storage

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

func New() *Storage {
	return &Storage{}
}

type Storage struct {
	mu   sync.Mutex
	kept []judgements.Snapshot
	err  error
}

var _ judgements.Storage = (*Storage)(nil)

// Every call fails with err from now on.
func (s *Storage) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.err = err
}

// Kept only once the fake transaction commits.
func (s *Storage) Save(ctx context.Context, judgement judgements.Judgement) error {
	s.mu.Lock()
	failed := s.err
	s.mu.Unlock()
	if failed != nil {
		return failed
	}
	cpatomic.OnCommit(ctx, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		saved := judgement.TakeSnapshot()
		if !slices.ContainsFunc(s.kept, func(kept judgements.Snapshot) bool { return kept.Report == saved.Report }) {
			s.kept = append(s.kept, saved)
		}
	})
	return nil
}

func (s *Storage) Kept() []judgements.Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.kept)
}

func (s *Storage) OfReport(_ context.Context, report judgements.ReportID) (judgements.Judgement, error) {
	return s.first(func(kept judgements.Snapshot) bool { return kept.Report == report })
}

func (s *Storage) OfMessage(_ context.Context, message messages.MessageID) (judgements.Judgement, error) {
	return s.first(func(kept judgements.Snapshot) bool {
		return message != judgements.NoMessage && kept.Message == message && kept.Judged
	})
}

func (s *Storage) first(match func(judgements.Snapshot) bool) (judgements.Judgement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return judgements.Judgement{}, s.err
	}
	found := slices.Clone(s.kept)
	found = slices.DeleteFunc(found, func(kept judgements.Snapshot) bool { return !match(kept) })
	if len(found) == 0 {
		return judgements.Judgement{}, judgements.ErrNotJudged
	}
	slices.SortStableFunc(found, func(a, b judgements.Snapshot) int { return a.At.Compare(b.At) })
	return found[0].Restore() //nolint:wrapcheck // the domain names it.
}

func (s *Storage) DeleteBefore(_ context.Context, cutoff time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return 0, s.err
	}
	before := len(s.kept)
	s.kept = slices.DeleteFunc(s.kept, func(kept judgements.Snapshot) bool { return kept.At.Before(cutoff) })
	return int64(before - len(s.kept)), nil
}

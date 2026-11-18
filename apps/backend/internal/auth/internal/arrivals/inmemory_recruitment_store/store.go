//go:build testing

package inmemory_recruitment_store

import (
	"context"
	"slices"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
)

func New() *Store {
	return &Store{}
}

type Store struct {
	mu   sync.Mutex
	kept []arrivals.RecruitmentSnapshot
}

var _ arrivals.Recruitments = (*Store)(nil)

func (s *Store) Seed(snapshots ...arrivals.RecruitmentSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.kept = append(s.kept, snapshots...)
}

func (s *Store) Kept() []arrivals.RecruitmentSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Clone(s.kept)
}

func (s *Store) Save(_ context.Context, saved ...arrivals.Recruitment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, recruitment := range saved {
		snapshot := recruitment.TakeSnapshot()
		s.kept = slices.DeleteFunc(s.kept, func(kept arrivals.RecruitmentSnapshot) bool { return kept.Arrival == snapshot.Arrival })
		s.kept = append(s.kept, snapshot)
	}
	return nil
}

func (s *Store) By(_ context.Context, recruiter arrivals.AccountID) ([]arrivals.Recruitment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	found := []arrivals.Recruitment{}
	for _, snapshot := range s.kept {
		if snapshot.Recruiter != recruiter || recruiter == (arrivals.AccountID{}) {
			continue
		}
		recruitment, err := snapshot.Restore()
		if err != nil {
			return nil, err //nolint:wrapcheck // a test store hands the rule's error on.
		}
		found = append(found, recruitment)
	}
	return found, nil
}

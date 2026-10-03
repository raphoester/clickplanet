//go:build testing

package inmemory_subscription_store

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
)

type Store struct {
	mu       sync.Mutex
	rows     map[subscriptions.AccountID]subscriptions.Subscription
	failWith error
}

var _ subscriptions.Store = (*Store)(nil)

func New() *Store {
	return &Store{rows: map[subscriptions.AccountID]subscriptions.Subscription{}}
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failWith = err
}

func (s *Store) Subscription(_ context.Context, account subscriptions.AccountID) (*subscriptions.Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	row, found := s.rows[account]
	if !found {
		return nil, subscriptions.ErrNoSubscription
	}
	return &row, nil
}

func (s *Store) SubscriptionsTo(_ context.Context, address subscriptions.Address) ([]*subscriptions.Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	found := []*subscriptions.Subscription{}
	for _, row := range s.rows {
		if row.Address == address {
			found = append(found, &row)
		}
	}
	return found, nil
}

func (s *Store) Save(_ context.Context, subscription *subscriptions.Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	s.rows[subscription.Account] = *subscription
	return nil
}

func (s *Store) Delete(_ context.Context, account subscriptions.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	delete(s.rows, account)
	return nil
}

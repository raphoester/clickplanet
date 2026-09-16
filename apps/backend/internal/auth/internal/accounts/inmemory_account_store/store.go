//go:build testing

// Package inmemory_account_store is accounts.Sessions in a map, for tests. It runs the same contract as postgres.
package inmemory_account_store

import (
	"context"
	"errors"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

var errTaken = errors.New("the token hash is taken")

type Store struct {
	mu       sync.Mutex
	sessions map[string]accounts.Session
	failWith error
}

var _ accounts.Sessions = (*Store)(nil)

func New() *Store {
	return &Store{sessions: map[string]accounts.Session{}}
}

// FailWith makes every call answer err, as a store that lost its database would.
func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failWith = err
}

func (s *Store) FindSession(_ context.Context, tokenHash []byte) (*accounts.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}

	session, found := s.sessions[string(tokenHash)]
	if !found {
		return nil, accounts.ErrSessionNotFound
	}
	return &session, nil
}

func (s *Store) CreateGuest(_ context.Context, session *accounts.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	if _, taken := s.sessions[string(session.TokenHash)]; taken {
		return errTaken
	}

	s.sessions[string(session.TokenHash)] = *session
	return nil
}

func (s *Store) SaveSession(_ context.Context, session *accounts.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	if _, found := s.sessions[string(session.TokenHash)]; !found {
		return accounts.ErrSessionNotFound
	}

	s.sessions[string(session.TokenHash)] = *session
	return nil
}

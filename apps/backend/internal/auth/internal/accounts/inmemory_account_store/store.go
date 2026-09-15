//go:build testing

// Package inmemory_account_store is accounts.Sessions in a map, for tests. It runs the same contract as postgres.
package inmemory_account_store

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

var (
	errTaken   = errors.New("the token hash is taken")
	errUnknown = errors.New("no session for this token hash")
)

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

func (s *Store) FindSession(_ context.Context, tokenHash []byte) (accounts.Session, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return accounts.Session{}, false, s.failWith
	}

	session, found := s.sessions[string(tokenHash)]
	return session, found, nil
}

func (s *Store) ExtendSession(_ context.Context, tokenHash []byte, expiresAt, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}

	session, found := s.sessions[string(tokenHash)]
	if !found {
		return errUnknown
	}

	session.ExtendedAt, session.ExpiresAt = now, expiresAt
	s.sessions[string(tokenHash)] = session
	return nil
}

func (s *Store) CreateGuest(_ context.Context, account uuid.UUID, tokenHash []byte, expiresAt, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}

	if _, taken := s.sessions[string(tokenHash)]; taken {
		return errTaken
	}

	s.sessions[string(tokenHash)] = accounts.Session{Account: account, ExtendedAt: now, ExpiresAt: expiresAt}
	return nil
}

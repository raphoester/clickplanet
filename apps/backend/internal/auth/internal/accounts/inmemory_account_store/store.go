//go:build testing

// Package inmemory_account_store is accounts.Store in maps, for tests. It runs the same contract as postgres.
package inmemory_account_store

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

var errTaken = errors.New("the token hash is taken")

type identityKey struct {
	provider string
	subject  string
}

type Store struct {
	mu         sync.Mutex
	lastSeen   map[uuid.UUID]time.Time
	identities map[identityKey]accounts.Identity
	sessions   map[string]accounts.Session
	failWith   error
}

var _ accounts.Store = (*Store)(nil)

func New() *Store {
	return &Store{
		lastSeen:   map[uuid.UUID]time.Time{},
		identities: map[identityKey]accounts.Identity{},
		sessions:   map[string]accounts.Session{},
	}
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
	session.Linked = len(s.identitiesOf(session.Account)) > 0
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

	s.lastSeen[session.Account] = session.ExtendedAt
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
	s.lastSeen[session.Account] = session.ExtendedAt
	return nil
}

func (s *Store) DeleteSession(_ context.Context, tokenHash []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}

	delete(s.sessions, string(tokenHash))
	return nil
}

func (s *Store) DeleteSessions(_ context.Context, account uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}

	s.deleteSessionsOf(account)
	return nil
}

func (s *Store) FindAccount(_ context.Context, account uuid.UUID) (*accounts.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	if _, found := s.lastSeen[account]; !found {
		return nil, accounts.ErrAccountNotFound
	}

	return &accounts.Account{ID: account, Identities: s.identitiesOf(account)}, nil
}

func (s *Store) FindIdentity(_ context.Context, provider string, subject string) (*accounts.Identity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}

	identity, found := s.identities[identityKey{provider: provider, subject: subject}]
	if !found {
		return nil, accounts.ErrIdentityNotFound
	}
	return &identity, nil
}

func (s *Store) SaveSignIn(_ context.Context, signIn accounts.SignIn) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	account := signIn.Session.Account
	if _, taken := s.sessions[string(signIn.Session.TokenHash)]; taken {
		return errTaken
	}
	if signIn.Identity != nil {
		if _, taken := s.identities[keyOf(signIn.Identity)]; taken {
			return accounts.ErrIdentityTaken
		}
	}

	if signIn.Identity != nil {
		s.identities[keyOf(signIn.Identity)] = *signIn.Identity
	}
	if signIn.Replaces != nil {
		delete(s.sessions, string(signIn.Replaces))
	}
	s.sessions[string(signIn.Session.TokenHash)] = *signIn.Session
	s.lastSeen[account] = signIn.Session.ExtendedAt
	return nil
}

func (s *Store) DeleteAccount(_ context.Context, account uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}

	s.deleteAccount(account)
	return nil
}

func (s *Store) PruneGuests(_ context.Context, idleSince time.Time, limit int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return 0, s.failWith
	}

	pruned := 0
	for account, seen := range s.lastSeen {
		if pruned == limit {
			break
		}
		if seen.Before(idleSince) && len(s.identitiesOf(account)) == 0 {
			s.deleteAccount(account)
			pruned++
		}
	}
	return pruned, nil
}

func (s *Store) deleteAccount(account uuid.UUID) {
	delete(s.lastSeen, account)
	for key, identity := range s.identities {
		if identity.Account == account {
			delete(s.identities, key)
		}
	}
	s.deleteSessionsOf(account)
}

func (s *Store) deleteSessionsOf(account uuid.UUID) {
	for hash, session := range s.sessions {
		if session.Account == account {
			delete(s.sessions, hash)
		}
	}
}

func (s *Store) identitiesOf(account uuid.UUID) []accounts.Identity {
	identities := []accounts.Identity{}
	for _, identity := range s.identities {
		if identity.Account == account {
			identities = append(identities, identity)
		}
	}
	slices.SortFunc(identities, func(a, b accounts.Identity) int { return a.LinkedAt.Compare(b.LinkedAt) })
	return identities
}

func keyOf(identity *accounts.Identity) identityKey {
	return identityKey{provider: identity.Provider, subject: identity.Subject}
}

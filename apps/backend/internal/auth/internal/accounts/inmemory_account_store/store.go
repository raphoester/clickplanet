//go:build testing

package inmemory_account_store

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

var errTaken = errors.New("the token hash is taken")

type identityKey struct {
	provider string
	subject  string
}

type Store struct {
	mu         sync.Mutex
	created    map[accounts.AccountID]time.Time
	lastSeen   map[accounts.AccountID]time.Time
	identities map[identityKey]accounts.Identity
	sessions   map[string]accounts.Session
	failWith   error
}

var _ accounts.Store = (*Store)(nil)

func New() *Store {
	return &Store{
		created:    map[accounts.AccountID]time.Time{},
		lastSeen:   map[accounts.AccountID]time.Time{},
		identities: map[identityKey]accounts.Identity{},
		sessions:   map[string]accounts.Session{},
	}
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failWith = err
}

func (s *Store) Session(_ context.Context, tokenHash accounts.TokenHash) (*accounts.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}

	session, found := s.sessions[string(tokenHash)]
	if !found {
		return nil, accounts.ErrSessionNotFound
	}
	return accounts.SessionOf(session.TokenHash(), session.Account(), len(s.identitiesOf(session.Account())) > 0,
		session.ExtendedAt(), session.ExpiresAt()), nil
}

func (s *Store) CreateGuest(_ context.Context, session *accounts.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	if _, taken := s.sessions[string(session.TokenHash())]; taken {
		return errTaken
	}

	s.created[session.Account()] = session.ExtendedAt()
	s.lastSeen[session.Account()] = session.ExtendedAt()
	s.sessions[string(session.TokenHash())] = *session
	return nil
}

func (s *Store) SaveSession(_ context.Context, session *accounts.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	if _, found := s.sessions[string(session.TokenHash())]; !found {
		return accounts.ErrSessionNotFound
	}

	s.sessions[string(session.TokenHash())] = *session
	s.lastSeen[session.Account()] = session.ExtendedAt()
	return nil
}

func (s *Store) DeleteSession(_ context.Context, tokenHash accounts.TokenHash) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}

	delete(s.sessions, string(tokenHash))
	return nil
}

func (s *Store) DeleteSessions(_ context.Context, account accounts.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}

	s.deleteSessionsOf(account)
	return nil
}

func (s *Store) Account(_ context.Context, account accounts.AccountID) (*accounts.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	if _, found := s.lastSeen[account]; !found {
		return nil, accounts.ErrAccountNotFound
	}

	return accounts.AccountOf(account, s.created[account], s.identitiesOf(account)), nil
}

func (s *Store) Identity(_ context.Context, provider string, subject string) (*accounts.Identity, error) {
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

func (s *Store) AccountOfEmail(_ context.Context, address string) (*accounts.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}

	holders := []accounts.Identity{}
	for _, identity := range s.identities {
		if identity.EmailVerified() && strings.EqualFold(identity.Email(), address) {
			holders = append(holders, identity)
		}
	}
	if len(holders) == 0 {
		return nil, accounts.ErrAccountNotFound
	}
	oldest := slices.MinFunc(holders, func(a, b accounts.Identity) int {
		return cmp.Or(a.LinkedAt().Compare(b.LinkedAt()), cmp.Compare(a.Provider(), b.Provider()))
	})
	return accounts.AccountOf(oldest.Account(), s.created[oldest.Account()], s.identitiesOf(oldest.Account())), nil
}

func (s *Store) SaveSignIn(_ context.Context, signIn accounts.SignIn) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	session, identity := signIn.Session(), signIn.Identity()
	account := session.Account()
	if _, taken := s.sessions[string(session.TokenHash())]; taken {
		return errTaken
	}
	if identity != nil {
		if _, taken := s.identities[keyOf(identity)]; taken {
			return accounts.ErrIdentityTaken
		}
	}

	if signIn.NewAccount() {
		s.created[account] = session.ExtendedAt()
	}
	if identity != nil {
		s.identities[keyOf(identity)] = *identity
	}
	if replaces := signIn.Replaces(); replaces != nil {
		delete(s.sessions, string(replaces))
	}
	s.sessions[string(session.TokenHash())] = *session
	s.lastSeen[account] = session.ExtendedAt()
	return nil
}

func (s *Store) DeleteAccount(_ context.Context, account accounts.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}

	s.deleteAccount(account)
	return nil
}

func (s *Store) PruneGuests(_ context.Context, idleSince time.Time, limit int) ([]accounts.AccountID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}

	var pruned []accounts.AccountID
	for account, seen := range s.lastSeen {
		if len(pruned) == limit {
			break
		}
		if seen.Before(idleSince) && len(s.identitiesOf(account)) == 0 {
			s.deleteAccount(account)
			pruned = append(pruned, account)
		}
	}
	return pruned, nil
}

func (s *Store) deleteAccount(account accounts.AccountID) {
	delete(s.created, account)
	delete(s.lastSeen, account)
	for key, identity := range s.identities {
		if identity.Account() == account {
			delete(s.identities, key)
		}
	}
	s.deleteSessionsOf(account)
}

func (s *Store) deleteSessionsOf(account accounts.AccountID) {
	for hash, session := range s.sessions {
		if session.Account() == account {
			delete(s.sessions, hash)
		}
	}
}

func (s *Store) identitiesOf(account accounts.AccountID) []accounts.Identity {
	identities := []accounts.Identity{}
	for _, identity := range s.identities {
		if identity.Account() == account {
			identities = append(identities, identity)
		}
	}
	slices.SortFunc(identities, func(a, b accounts.Identity) int {
		return cmp.Or(a.LinkedAt().Compare(b.LinkedAt()), cmp.Compare(a.Provider(), b.Provider()))
	})
	return identities
}

func keyOf(identity *accounts.Identity) identityKey {
	return identityKey{provider: identity.Provider(), subject: identity.Subject()}
}

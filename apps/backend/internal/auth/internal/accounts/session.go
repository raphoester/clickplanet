package accounts

import (
	"fmt"
	"time"
)

type Session struct {
	TokenHash  TokenHash
	Account    AccountID
	Linked     bool
	ExtendedAt time.Time
	ExpiresAt  time.Time
}

func GuestSession(account AccountID, token *Token, lifetime Lifetime, now time.Time) *Session {
	return newSession(account, token, false, lifetime, now)
}

func LinkedSession(account AccountID, token *Token, lifetime Lifetime, now time.Time) *Session {
	return newSession(account, token, true, lifetime, now)
}

func newSession(account AccountID, token *Token, linked bool, lifetime Lifetime, now time.Time) *Session {
	session := &Session{TokenHash: token.Hash, Account: account, Linked: linked, ExtendedAt: now}
	session.ExpiresAt = now.Add(session.ttl(lifetime))
	return session
}

func (s *Session) ExpiryError(now time.Time) error {
	if !now.Before(s.ExpiresAt) {
		return fmt.Errorf("%w at %s", ErrSessionExpired, s.ExpiresAt.Format(time.RFC3339))
	}
	return nil
}

func (s *Session) Extendable(now time.Time, lifetime Lifetime) bool {
	return now.Sub(s.ExtendedAt) >= lifetime.ExtendEvery
}

func (s *Session) Extended(now time.Time, lifetime Lifetime) *Session {
	extended := *s
	extended.ExtendedAt = now
	extended.ExpiresAt = now.Add(s.ttl(lifetime))
	return &extended
}

func (s *Session) ttl(lifetime Lifetime) time.Duration {
	if s.Linked {
		return lifetime.LinkedTTL
	}
	return lifetime.GuestTTL
}

func (s *Session) Cookie(token *Token, now time.Time) string {
	return Cookie(CookieName, token.Value, s.ExpiresAt, now)
}

type Lifetime struct {
	GuestTTL time.Duration

	LinkedTTL time.Duration

	ExtendEvery time.Duration
}

const (
	defaultGuestTTL    = 90 * 24 * time.Hour
	defaultLinkedTTL   = 30 * 24 * time.Hour
	defaultExtendEvery = 24 * time.Hour
)

func (l Lifetime) WithDefaults() Lifetime {
	if l.GuestTTL <= 0 {
		l.GuestTTL = defaultGuestTTL
	}
	if l.LinkedTTL <= 0 {
		l.LinkedTTL = defaultLinkedTTL
	}
	if l.ExtendEvery <= 0 {
		l.ExtendEvery = defaultExtendEvery
	}
	return l
}

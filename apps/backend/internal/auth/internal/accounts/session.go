package accounts

import (
	"fmt"
	"time"
)

type Session struct {
	tokenHash  TokenHash
	account    AccountID
	linked     bool
	extendedAt time.Time
	expiresAt  time.Time
}

func GuestSession(account AccountID, token *Token, lifetime Lifetime, now time.Time) *Session {
	return newSession(account, token, false, lifetime, now)
}

func LinkedSession(account AccountID, token *Token, lifetime Lifetime, now time.Time) *Session {
	return newSession(account, token, true, lifetime, now)
}

func newSession(account AccountID, token *Token, linked bool, lifetime Lifetime, now time.Time) *Session {
	session := &Session{tokenHash: token.hash, account: account, linked: linked, extendedAt: now}
	session.expiresAt = now.Add(session.ttl(lifetime))
	return session
}

func SessionOf(tokenHash TokenHash, account AccountID, linked bool, extendedAt time.Time, expiresAt time.Time) *Session {
	return &Session{tokenHash: tokenHash, account: account, linked: linked, extendedAt: extendedAt, expiresAt: expiresAt}
}

func (s *Session) TokenHash() TokenHash {
	return s.tokenHash
}

func (s *Session) Account() AccountID {
	return s.account
}

func (s *Session) Linked() bool {
	return s.linked
}

func (s *Session) ExtendedAt() time.Time {
	return s.extendedAt
}

func (s *Session) ExpiresAt() time.Time {
	return s.expiresAt
}

func (s *Session) ExpiryError(now time.Time) error {
	if !now.Before(s.expiresAt) {
		return fmt.Errorf("%w at %s", ErrSessionExpired, s.expiresAt.Format(time.RFC3339))
	}
	return nil
}

func (s *Session) Extendable(now time.Time, lifetime Lifetime) bool {
	return now.Sub(s.extendedAt) >= lifetime.ExtendEvery
}

func (s *Session) Extended(now time.Time, lifetime Lifetime) *Session {
	extended := *s
	extended.extendedAt = now
	extended.expiresAt = now.Add(s.ttl(lifetime))
	return &extended
}

func (s *Session) ttl(lifetime Lifetime) time.Duration {
	if s.linked {
		return lifetime.LinkedTTL
	}
	return lifetime.GuestTTL
}

func (s *Session) Cookie(token *Token, now time.Time) string {
	return Cookie(CookieName, token.value, s.expiresAt, now)
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

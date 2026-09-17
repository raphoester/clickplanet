// Package accounts is who a browser is: an account, and the session its cookie holds.
package accounts

import (
	"fmt"
	"time"
)

// Session is one browser's hold on an account, found by the hash of its cookie's token.
type Session struct {
	TokenHash TokenHash
	Account   AccountID
	// Linked is whether the account has a provider, which sets how long the session lasts. The store reads it, never writes it.
	Linked     bool
	ExtendedAt time.Time
	ExpiresAt  time.Time
}

// GuestSession opens a new account's first session.
func GuestSession(account AccountID, token *Token, lifetime Lifetime, now time.Time) *Session {
	return newSession(account, token, false, lifetime, now)
}

// LinkedSession opens a session on an account that has a provider, as a sign-in does.
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

// Extension is the session's expiry as using it at now would move it. Nothing changes until Extend is given it.
type Extension struct {
	extendedAt time.Time
	expiresAt  time.Time
	due        bool
}

// Due is whether the last extension is old enough to extend again, so a busy player is not a write per visit.
func (e Extension) Due() bool {
	return e.due
}

func (s *Session) Extension(now time.Time, lifetime Lifetime) Extension {
	return Extension{
		extendedAt: now,
		expiresAt:  now.Add(s.ttl(lifetime)),
		due:        now.Sub(s.ExtendedAt) >= lifetime.ExtendEvery,
	}
}

// Extend moves the expiry to the extension's, whether or not it was due: the caller decides that from Due.
func (s *Session) Extend(extension Extension) {
	s.ExtendedAt = extension.extendedAt
	s.ExpiresAt = extension.expiresAt
}

func (s *Session) ttl(lifetime Lifetime) time.Duration {
	if s.Linked {
		return lifetime.LinkedTTL
	}
	return lifetime.GuestTTL
}

// Cookie keeps token in the browser for as long as the session lives.
func (s *Session) Cookie(token *Token, now time.Time) string {
	return Cookie(CookieName, token.Value, s.ExpiresAt, now)
}

// Lifetime is how long a session lasts, and how often using it pushes that out.
type Lifetime struct {
	// A guest idle this long loses its cookie (default 90 days, the guest prune window).
	GuestTTL time.Duration

	// A signed-in account idle this long loses its cookie (default 30 days).
	LinkedTTL time.Duration

	// A session is extended at most this often (default 24h).
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

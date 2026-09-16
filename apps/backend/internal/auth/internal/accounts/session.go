// Package accounts is who a browser is: an account, and the session its cookie holds.
package accounts

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Session is one browser's hold on an account, found by the hash of its cookie's token.
type Session struct {
	TokenHash  []byte
	Account    uuid.UUID
	ExtendedAt time.Time
	ExpiresAt  time.Time
}

// StartGuest opens a new account's first session.
func StartGuest(account uuid.UUID, token *Token, lifetime Lifetime, now time.Time) *Session {
	return &Session{
		TokenHash:  token.Hash,
		Account:    account,
		ExtendedAt: now,
		ExpiresAt:  now.Add(lifetime.GuestTTL),
	}
}

func (s *Session) CheckLive(now time.Time) error {
	if !now.Before(s.ExpiresAt) {
		return fmt.Errorf("%w at %s", ErrSessionExpired, s.ExpiresAt.Format(time.RFC3339))
	}
	return nil
}

// ExtendIfDue moves the expiry out when the last extension is old enough, so a busy player is not a write per visit.
func (s *Session) ExtendIfDue(now time.Time, lifetime Lifetime) bool {
	if now.Sub(s.ExtendedAt) < lifetime.ExtendEvery {
		return false
	}

	s.ExtendedAt = now
	s.ExpiresAt = now.Add(lifetime.GuestTTL)
	return true
}

// Cookie keeps token in the browser for as long as the session lives.
func (s *Session) Cookie(token *Token, now time.Time) string {
	return setCookie(token.Value, s.ExpiresAt, now)
}

// Lifetime is how long a session lasts, and how often using it pushes that out.
type Lifetime struct {
	// A guest idle this long loses its cookie (default 90 days, the guest prune window).
	GuestTTL time.Duration

	// A session is extended at most this often (default 24h).
	ExtendEvery time.Duration
}

const (
	defaultGuestTTL    = 90 * 24 * time.Hour
	defaultExtendEvery = 24 * time.Hour
)

func (l Lifetime) WithDefaults() Lifetime {
	if l.GuestTTL <= 0 {
		l.GuestTTL = defaultGuestTTL
	}
	if l.ExtendEvery <= 0 {
		l.ExtendEvery = defaultExtendEvery
	}
	return l
}

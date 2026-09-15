// Package accounts is who a browser is: an account, and the session its cookie holds.
package accounts

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Session is one signed-in browser, found by the hash of its cookie's token.
type Session struct {
	Account    uuid.UUID
	ExtendedAt time.Time
	ExpiresAt  time.Time
}

// Lifetime is how long a session lasts, and how often using it pushes that out.
type Lifetime struct {
	// A guest idle this long loses its cookie (default 90 days, the guest prune window).
	GuestTTL time.Duration

	// A session is extended at most this often, so a busy player is not a write per mint (default 24h).
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

func (s Session) Live(now time.Time) bool {
	return now.Before(s.ExpiresAt)
}

func (l Lifetime) ExtensionDue(session Session, now time.Time) bool {
	return now.Sub(session.ExtendedAt) >= l.ExtendEvery
}

func (l Lifetime) ExpiryFrom(now time.Time) time.Time {
	return now.Add(l.GuestTTL)
}

// Token is a session's secret: the value goes in the cookie, the hash in the table.
type Token struct {
	Value string
	Hash  []byte
}

const tokenBytes = 32

func NewToken() (Token, error) {
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return Token{}, fmt.Errorf("failed to read random bytes: %w", err)
	}

	value := base64.RawURLEncoding.EncodeToString(raw)
	return Token{Value: value, Hash: HashOf(value)}, nil
}

func HashOf(value string) []byte {
	sum := sha256.Sum256([]byte(value))
	return sum[:]
}

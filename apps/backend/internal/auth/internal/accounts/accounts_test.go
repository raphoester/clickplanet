package accounts_test

import (
	"crypto/sha256"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

var (
	now      = time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	lifetime = accounts.Lifetime{}.WithDefaults()
	account  = uuid.UUID{15: 1}
)

func TestTheTokenIsFoundAmongOtherCookies(t *testing.T) {
	token, err := accounts.TokenFromCookies("theme=dark; cp_sid=abc123; lang=fr")

	require.NoError(t, err)
	assert.Equal(t, accounts.TokenOf("abc123"), token)
}

func TestNoTokenInAHeaderWithoutTheCookie(t *testing.T) {
	for name, header := range map[string]string{
		"empty":        "",
		"other cookie": "theme=dark",
		"empty value":  "cp_sid=",
		"garbage":      ";;;=",
	} {
		t.Run(name, func(t *testing.T) {
			token, err := accounts.TokenFromCookies(header)

			require.ErrorIs(t, err, accounts.ErrNoSessionCookie)
			assert.Nil(t, token)
		})
	}
}

func TestATokensHashIsTheSHA256OfItsValue(t *testing.T) {
	sum := sha256.Sum256([]byte("abc123"))

	assert.Equal(t, &accounts.Token{Value: "abc123", Hash: sum[:]}, accounts.TokenOf("abc123"))
}

func TestAGuestStartsWithAFullLifetime(t *testing.T) {
	token := accounts.TokenOf("abc123")

	session := accounts.StartGuest(account, token, lifetime, now)

	assert.Equal(t, &accounts.Session{
		TokenHash: token.Hash, Account: account, ExtendedAt: now, ExpiresAt: now.Add(90 * 24 * time.Hour),
	}, session)
}

func TestASessionEndsAtItsExpiry(t *testing.T) {
	session := accounts.StartGuest(account, accounts.TokenOf("abc123"), lifetime, now)

	require.NoError(t, session.CheckLive(now.Add(90*24*time.Hour-time.Second)))
	assert.ErrorIs(t, session.CheckLive(now.Add(90*24*time.Hour)), accounts.ErrSessionExpired)
}

func TestASessionIsExtendedAtMostOnceAnInterval(t *testing.T) {
	session := accounts.StartGuest(account, accounts.TokenOf("abc123"), lifetime, now)

	assert.False(t, session.ExtendIfDue(now.Add(23*time.Hour), lifetime))
	assert.Equal(t, now.Add(90*24*time.Hour), session.ExpiresAt)

	later := now.Add(24 * time.Hour)
	assert.True(t, session.ExtendIfDue(later, lifetime))
	assert.Equal(t, later, session.ExtendedAt)
	assert.Equal(t, later.Add(90*24*time.Hour), session.ExpiresAt)
}

func TestTheCookieLivesAsLongAsTheSessionAndStaysOnThisSite(t *testing.T) {
	session := accounts.StartGuest(account, accounts.TokenOf("abc123"), accounts.Lifetime{GuestTTL: time.Hour}.WithDefaults(), now)

	cookie, err := http.ParseSetCookie(session.Cookie(accounts.TokenOf("abc123"), now))
	require.NoError(t, err)

	assert.Equal(t, "cp_sid", cookie.Name)
	assert.Equal(t, "abc123", cookie.Value)
	assert.Equal(t, "/", cookie.Path)
	assert.Equal(t, 3600, cookie.MaxAge)
	assert.True(t, cookie.HttpOnly)
	assert.True(t, cookie.Secure)
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
	assert.Empty(t, cookie.Domain, "host-only: the cookie never reaches another subdomain")
}

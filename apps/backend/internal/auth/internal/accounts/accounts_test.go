package accounts_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

var now = time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)

func TestTheTokenIsFoundAmongOtherCookies(t *testing.T) {
	token, ok := accounts.TokenFrom("theme=dark; cp_sid=abc123; lang=fr")

	require.True(t, ok)
	assert.Equal(t, "abc123", token)
}

func TestNoTokenInAHeaderWithoutTheCookie(t *testing.T) {
	for name, header := range map[string]string{
		"empty":        "",
		"other cookie": "theme=dark",
		"empty value":  "cp_sid=",
		"garbage":      ";;;=",
	} {
		t.Run(name, func(t *testing.T) {
			_, ok := accounts.TokenFrom(header)
			assert.False(t, ok)
		})
	}
}

func TestTheSetCookieIsOnlyForHTTPAndSentByThisSiteAlone(t *testing.T) {
	header := accounts.SetCookie("abc123", now.Add(time.Hour), now)

	cookie, err := http.ParseSetCookie(header)
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

func TestTwoTokensDifferAndOnlyTheirHashesMatch(t *testing.T) {
	first, err := accounts.NewToken()
	require.NoError(t, err)
	second, err := accounts.NewToken()
	require.NoError(t, err)

	assert.NotEqual(t, first.Value, second.Value)
	assert.Equal(t, first.Hash, accounts.HashOf(first.Value))
	assert.NotEqual(t, first.Hash, accounts.HashOf(second.Value))
}

func TestASessionIsExtendedAtMostOnceAnInterval(t *testing.T) {
	lifetime := accounts.Lifetime{}.WithDefaults()
	session := accounts.Session{ExtendedAt: now, ExpiresAt: lifetime.ExpiryFrom(now)}

	assert.False(t, lifetime.ExtensionDue(session, now.Add(23*time.Hour)))
	assert.True(t, lifetime.ExtensionDue(session, now.Add(24*time.Hour)))
}

func TestASessionEndsAtItsExpiry(t *testing.T) {
	lifetime := accounts.Lifetime{}.WithDefaults()
	session := accounts.Session{ExtendedAt: now, ExpiresAt: lifetime.ExpiryFrom(now)}

	assert.True(t, session.Live(now.Add(90*24*time.Hour-time.Second)))
	assert.False(t, session.Live(now.Add(90*24*time.Hour)))
}

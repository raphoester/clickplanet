package random_token_generator_test

import (
	"encoding/base64"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/random_token_generator"
)

func cookieValue(t *testing.T, token *accounts.Token) string {
	t.Helper()

	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	session := accounts.GuestSession(accounts.AccountID{15: 1}, token, accounts.Lifetime{}.WithDefaults(), now)
	cookie, err := http.ParseSetCookie(session.Cookie(token, now))
	require.NoError(t, err)
	return cookie.Value
}

func TestATokenIs32RandomBytesWithItsHash(t *testing.T) {
	first, err := random_token_generator.Generator{}.NewToken()
	require.NoError(t, err)
	second, err := random_token_generator.Generator{}.NewToken()
	require.NoError(t, err)

	value := cookieValue(t, first)
	raw, err := base64.RawURLEncoding.DecodeString(value)
	require.NoError(t, err)
	assert.Len(t, raw, 32)
	assert.NotEqual(t, value, cookieValue(t, second))
	assert.Equal(t, accounts.TokenOf(value), first)
}

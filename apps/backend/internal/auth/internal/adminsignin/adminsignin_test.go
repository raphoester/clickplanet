package adminsignin_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/adminsignin"
)

var (
	start = time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	ada   = accounts.AccountID{15: 1}
)

func TestAnOperatorIsIssuedASessionOfTheLifetimeAndNoMore(t *testing.T) {
	issuer := adminsignin.NewIssuer(&accounts.SequentialTokens{}, adminsignin.DefaultTTL)

	session, err := issuer.Session(adminsignin.CandidateOf(ada, true), start)
	require.NoError(t, err)

	assert.Equal(t, adminsignin.Snapshot{
		TokenHash: accounts.TokenOf("token-1").Hash(), Account: ada, CreatedAt: start, ExpiresAt: start.Add(12 * time.Hour),
	}, session.TakeSnapshot())

	cookie, err := http.ParseSetCookie(session.Cookie(start))
	require.NoError(t, err)
	assert.Equal(t, adminsignin.CookieName, cookie.Name)
	assert.Equal(t, "token-1", cookie.Value)
	assert.Equal(t, int((12 * time.Hour).Seconds()), cookie.MaxAge)
	assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite, "the browser sends it on no call another site starts")
	assert.True(t, cookie.HttpOnly)
	assert.True(t, cookie.Secure)
}

func TestSomebodyWhoDoesNotOperateIsIssuedNothing(t *testing.T) {
	_, err := adminsignin.NewIssuer(&accounts.SequentialTokens{}, adminsignin.DefaultTTL).
		Session(adminsignin.CandidateOf(ada, false), start)

	require.ErrorIs(t, err, adminsignin.ErrNotAnOperator)
}

func TestTheAdminCookiesAreApartFromTheGamesOnes(t *testing.T) {
	session, err := http.ParseSetCookie(adminsignin.ExpiredCookie())
	require.NoError(t, err)
	flow, err := http.ParseSetCookie(adminsignin.ExpiredFlowCookie())
	require.NoError(t, err)

	assert.Equal(t, "cp_admin", session.Name)
	assert.Equal(t, "cp_admin_oauth", flow.Name, "a game sign-in in progress is never overwritten")
	assert.Negative(t, session.MaxAge)
}

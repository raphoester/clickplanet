package accounts_test

import (
	"crypto/sha256"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

var (
	now      = time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	lifetime = accounts.Lifetime{}.WithDefaults()
	account  = accounts.AccountID{15: 1}
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

	session := accounts.GuestSession(account, token, lifetime, now)

	assert.Equal(t, &accounts.Session{
		TokenHash: token.Hash, Account: account, ExtendedAt: now, ExpiresAt: now.Add(90 * 24 * time.Hour),
	}, session)
}

func TestASessionEndsAtItsExpiry(t *testing.T) {
	session := accounts.GuestSession(account, accounts.TokenOf("abc123"), lifetime, now)

	require.NoError(t, session.ExpiryError(now.Add(90*24*time.Hour-time.Second)))
	assert.ErrorIs(t, session.ExpiryError(now.Add(90*24*time.Hour)), accounts.ErrSessionExpired)
}

func TestASessionIsExtendableAtMostOnceAnInterval(t *testing.T) {
	session := accounts.GuestSession(account, accounts.TokenOf("abc123"), lifetime, now)

	assert.False(t, session.Extendable(now.Add(23*time.Hour), lifetime))
	assert.True(t, session.Extendable(now.Add(24*time.Hour), lifetime))
}

func TestAnExtendedSessionIsACopyAndTheSessionIsLeftAsItWas(t *testing.T) {
	session := accounts.GuestSession(account, accounts.TokenOf("abc123"), lifetime, now)
	later := now.Add(24 * time.Hour)

	extended := session.Extended(later, lifetime)

	assert.Equal(t, later, extended.ExtendedAt)
	assert.Equal(t, later.Add(90*24*time.Hour), extended.ExpiresAt)
	assert.Equal(t, now, session.ExtendedAt)
	assert.Equal(t, now.Add(90*24*time.Hour), session.ExpiresAt)
}

func TestTheCookieLivesAsLongAsTheSessionAndStaysOnThisSite(t *testing.T) {
	session := accounts.GuestSession(account, accounts.TokenOf("abc123"), accounts.Lifetime{GuestTTL: time.Hour}.WithDefaults(), now)

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

func TestALinkedSessionLastsTheLinkedLifetimeAndExtendsByIt(t *testing.T) {
	session := accounts.LinkedSession(account, accounts.TokenOf("abc123"), lifetime, now)
	assert.Equal(t, now.Add(30*24*time.Hour), session.ExpiresAt)

	later := now.Add(24 * time.Hour)
	assert.Equal(t, later.Add(30*24*time.Hour), session.Extended(later, lifetime).ExpiresAt)
}

func TestAGuestSessionExtendsByTheLinkedLifetimeOnceItsAccountIsLinked(t *testing.T) {
	session := accounts.GuestSession(account, accounts.TokenOf("abc123"), lifetime, now)
	session.Linked = true

	later := now.Add(24 * time.Hour)
	assert.Equal(t, later.Add(30*24*time.Hour), session.Extended(later, lifetime).ExpiresAt)
}

func TestClearingTheCookieExpiresItWithTheSameAttributes(t *testing.T) {
	cookie, err := http.ParseSetCookie(accounts.ExpiredSessionCookie())
	require.NoError(t, err)

	assert.Equal(t, "cp_sid", cookie.Name)
	assert.Empty(t, cookie.Value)
	assert.Negative(t, cookie.MaxAge)
	assert.Equal(t, "/", cookie.Path)
	assert.True(t, cookie.HttpOnly)
	assert.True(t, cookie.Secure)
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
}

func TestAnUnverifiedEmailIsNeverKept(t *testing.T) {
	for name, claim := range map[string]accounts.Claim{
		"unverified":         {Subject: "user", Email: "a@example.com"},
		"verified with none": {Subject: "user", EmailVerified: true},
	} {
		t.Run(name, func(t *testing.T) {
			identity := accounts.NewIdentity("discord", claim, account, now)

			assert.Equal(t, &accounts.Identity{Provider: "discord", Subject: "user", Account: account, LinkedAt: now}, identity)
		})
	}
}

func TestAVerifiedEmailIsKept(t *testing.T) {
	identity := accounts.NewIdentity("google", accounts.Claim{Subject: "user", Email: "a@example.com", EmailVerified: true}, account, now)

	assert.Equal(t, "a@example.com", identity.Email)
	assert.True(t, identity.EmailVerified)
}

func TestAKnownIdentitySignsInWhateverTheBrowserIsOn(t *testing.T) {
	known := &accounts.Identity{Provider: "google", Subject: "user", Account: accounts.AccountID{15: 2}}

	assert.Equal(t, accounts.SignedIn, accounts.OutcomeOf(nil, known, "google"))
	assert.Equal(t, accounts.SignedIn, accounts.OutcomeOf(&accounts.Account{ID: account}, known, "google"))
}

func TestANewIdentityLinksToTheAccountTheBrowserIsOn(t *testing.T) {
	guest := &accounts.Account{ID: account}
	linkedElsewhere := &accounts.Account{ID: account, Identities: []accounts.Identity{{Provider: "discord"}}}

	assert.Equal(t, accounts.Linked, accounts.OutcomeOf(guest, nil, "google"))
	assert.Equal(t, accounts.Linked, accounts.OutcomeOf(linkedElsewhere, nil, "google"))
}

func TestANewIdentityWithNowhereToGoCreatesAnAccount(t *testing.T) {
	sameProvider := &accounts.Account{ID: account, Identities: []accounts.Identity{{Provider: "google", Subject: "someone else"}}}

	assert.Equal(t, accounts.Created, accounts.OutcomeOf(nil, nil, "google"))
	assert.Equal(t, accounts.Created, accounts.OutcomeOf(sameProvider, nil, "google"), "one account holds one user per provider")
}

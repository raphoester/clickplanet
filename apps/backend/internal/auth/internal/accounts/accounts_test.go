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

	for name, current := range map[string]*accounts.Account{"no account": nil, "another account": {ID: account}} {
		t.Run(name, func(t *testing.T) {
			outcome, err := accounts.OutcomeOf(accounts.IntentSignIn, current, known, "google")

			require.NoError(t, err)
			assert.Equal(t, accounts.SignedIn, outcome)
		})
	}
}

func TestANewIdentityLinksToTheAccountTheBrowserIsOn(t *testing.T) {
	for _, intent := range []accounts.Intent{accounts.IntentSignIn, accounts.IntentLink} {
		for name, current := range map[string]*accounts.Account{
			"guest":                   {ID: account},
			"linked to another place": {ID: account, Identities: []accounts.Identity{{Provider: "discord"}}},
		} {
			t.Run(name, func(t *testing.T) {
				outcome, err := accounts.OutcomeOf(intent, current, nil, "google")

				require.NoError(t, err)
				assert.Equal(t, accounts.Linked, outcome)
			})
		}
	}
}

func TestANewIdentityWithNowhereToGoCreatesAnAccount(t *testing.T) {
	for name, current := range map[string]*accounts.Account{
		"no account":    nil,
		"same provider": {ID: account, Identities: []accounts.Identity{{Provider: "google", Subject: "someone else"}}},
	} {
		t.Run(name, func(t *testing.T) {
			outcome, err := accounts.OutcomeOf(accounts.IntentSignIn, current, nil, "google")

			require.NoError(t, err)
			assert.Equal(t, accounts.Created, outcome)
		})
	}
}

func TestLinkingAnIdentityTheAccountAlreadyHasChangesNothing(t *testing.T) {
	current := &accounts.Account{ID: account, Identities: []accounts.Identity{{Provider: "google", Subject: "user", Account: account}}}

	outcome, err := accounts.OutcomeOf(accounts.IntentLink, current, &current.Identities[0], "google")

	require.NoError(t, err)
	assert.Equal(t, accounts.SignedIn, outcome, "the browser stays on its account")
}

func TestALinkIsRefusedRatherThanLeaveTheAccount(t *testing.T) {
	onDiscord := &accounts.Account{ID: account, Identities: []accounts.Identity{{Provider: "discord", Subject: "d", Account: account}}}
	elsewhere := &accounts.Identity{Provider: "google", Subject: "user", Account: accounts.AccountID{15: 2}}
	sameProvider := &accounts.Account{ID: account, Identities: []accounts.Identity{{Provider: "google", Subject: "someone else", Account: account}}}

	for name, tc := range map[string]struct {
		current *accounts.Account
		known   *accounts.Identity
		want    error
	}{
		"identity on another account": {current: onDiscord, known: elsewhere, want: accounts.ErrIdentityLinkedElsewhere},
		"provider already linked":     {current: sameProvider, want: accounts.ErrProviderAlreadyLinked},
		"no account to link to":       {want: accounts.ErrNoAccount},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := accounts.OutcomeOf(accounts.IntentLink, tc.current, tc.known, "google")

			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestAnAccountIDIsAUUIDAndNeverTheNilOne(t *testing.T) {
	id, err := accounts.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	require.NoError(t, err)
	assert.Equal(t, "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", id.String())

	for _, value := range []string{"", "not-a-uuid", "00000000-0000-0000-0000-000000000000"} {
		_, err := accounts.AccountIDOf(value)
		require.ErrorIs(t, err, accounts.ErrInvalidAccount, "%q", value)
	}
}

package signin_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
)

func challenge(t *testing.T, intent accounts.Intent, account accounts.AccountID) *signin.Challenge {
	t.Helper()

	challenge, err := signin.NewChallenge("player@example.com", intent, account, &signin.SequentialSecrets{}, &signin.SequentialCodes{}, now)
	require.NoError(t, err)
	return challenge
}

func TestAnythingButOneBareAddressIsRefused(t *testing.T) {
	for _, raw := range []string{
		"",
		"player",
		"player@",
		"@example.com",
		"Player <player@example.com>",
		"player@example.com (comment)",
		`"player one"@example.com`,
		"player@localhost",
		"player@[192.0.2.1]",
		"player@example..com",
		"player@-example.com",
		"player@example.com, other@example.com",
		strings.Repeat("a", 250) + "@example.com",
	} {
		t.Run(raw, func(t *testing.T) {
			_, err := signin.AddressOf(raw)

			assert.ErrorIs(t, err, signin.ErrAddressInvalid)
		})
	}
}

func TestOnlyTheCodeSentPassesWhileTheChallengeLives(t *testing.T) {
	c := challenge(t, accounts.IntentSignIn, accounts.AccountID{})

	require.NoError(t, c.CodeError("000001", now.Add(9*time.Minute)))
	assert.ErrorIs(t, c.CodeError("000002", now), signin.ErrWrongCode)
	assert.ErrorIs(t, c.CodeError("", now), signin.ErrWrongCode)
	assert.ErrorIs(t, c.CodeError("000001", now.Add(10*time.Minute)), signin.ErrFlowInvalid, "a lapsed challenge is started again")
}

func TestAnEmailLinkMustEndOnTheAccountItStartedOn(t *testing.T) {
	c := challenge(t, accounts.IntentLink, accounts.AccountID{15: 7})

	require.NoError(t, c.AccountError(accounts.AccountOf(accounts.AccountID{15: 7}, time.Time{}, nil)))
	assert.ErrorIs(t, c.AccountError(accounts.AccountOf(accounts.AccountID{15: 8}, time.Time{}, nil)), signin.ErrFlowInvalid)
	assert.ErrorIs(t, c.AccountError(nil), signin.ErrFlowInvalid)
	require.NoError(t, challenge(t, accounts.IntentSignIn, accounts.AccountID{}).AccountError(nil))
}

func TestTheChallengeCookieLivesAsLongAsTheChallenge(t *testing.T) {
	cookie, err := http.ParseSetCookie(challenge(t, accounts.IntentSignIn, accounts.AccountID{}).Cookie("sealed", now))
	require.NoError(t, err)

	assert.Equal(t, "cp_email", cookie.Name)
	assert.Equal(t, 600, cookie.MaxAge)
	assert.True(t, cookie.HttpOnly)
	assert.True(t, cookie.Secure)

	cleared, err := http.ParseSetCookie(signin.ExpiredChallengeCookie())
	require.NoError(t, err)
	assert.Equal(t, "cp_email", cleared.Name)
	assert.Negative(t, cleared.MaxAge)
}

func TestTheLetterCarriesTheCodeInEveryPart(t *testing.T) {
	letter := signin.CodeLetter("123456")

	assert.Equal(t, "Your ClickPlanet code: 123456", letter.Subject())
	assert.Contains(t, letter.Text(), "123456")
	assert.Contains(t, letter.HTML(), "123456")
	assert.Contains(t, letter.Text(), "10 minutes")
}

func TestTheOfferListsTheProvidersThenEmail(t *testing.T) {
	providers := signin.Providers{signin.Google: signin.NewFakeProvider(signin.Google), signin.Discord: signin.NewFakeProvider(signin.Discord)}

	assert.Equal(t, []string{"discord", "google", "email"}, signin.NewOffer(providers, true).Names())
	assert.Equal(t, []string{"discord", "google"}, signin.NewOffer(providers, false).Names())
	assert.Equal(t, []string{"email"}, signin.NewOffer(signin.Providers{}, true).Names())
	assert.Empty(t, signin.NewOffer(signin.Providers{}, false).Names())
}

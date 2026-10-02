package signin_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/inmemory_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/aes_flow_sealer"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func challenges(t *testing.T, offered bool) *signin.Challenges {
	t.Helper()

	sealer, err := aes_flow_sealer.New(bytes.Repeat([]byte{1}, 32))
	require.NoError(t, err)
	guesses := cpratelimit.New("guesses", cpratelimit.Config{Burst: signin.MaxAttempts, PerSecond: 1 / signin.ChallengeTTL.Seconds()},
		cptime.NewFixedClock(now))
	return signin.NewChallenges(offered, &signin.SequentialSecrets{}, &signin.SequentialCodes{}, sealer, guesses)
}

func cookieHeaderOf(t *testing.T, setCookie string) string {
	t.Helper()

	cookie, err := http.ParseSetCookie(setCookie)
	require.NoError(t, err)
	return cookie.Name + "=" + cookie.Value
}

func TestChallengesAreOffWhileEmailIsNotOffered(t *testing.T) {
	assert.True(t, challenges(t, false).Off())
	assert.False(t, challenges(t, true).Off())
}

func TestAnIssuedChallengeComesBackInItsCookie(t *testing.T) {
	c := challenges(t, true)

	issued, setCookie, err := c.Issued("player@example.com", accounts.IntentLink, accounts.AccountID{15: 7}, now)
	require.NoError(t, err)
	opened, err := c.Opened("cp_sid=other; " + cookieHeaderOf(t, setCookie))

	require.NoError(t, err)
	assert.Equal(t, issued, opened)
	assert.Equal(t, "000001", opened.Code)
}

func TestNoOrAForgedChallengeCookieIsStartedAgain(t *testing.T) {
	for _, header := range []string{"", "cp_sid=guest", "cp_email=forged"} {
		_, err := challenges(t, true).Opened(header)

		assert.ErrorIs(t, err, signin.ErrFlowInvalid, header)
	}
}

func TestEveryGuessSpendsOneAndTheRightOneTooOnceTheyAreGone(t *testing.T) {
	c := challenges(t, true)
	challenge, _, err := c.Issued("player@example.com", accounts.IntentSignIn, accounts.AccountID{}, now)
	require.NoError(t, err)

	for range signin.MaxAttempts - 1 {
		require.ErrorIs(t, c.Guess(challenge, "999999", now), signin.ErrWrongCode)
	}
	require.NoError(t, c.Guess(challenge, "000001", now))

	assert.ErrorIs(t, c.Guess(challenge, "000001", now), signin.ErrFlowInvalid)
}

func TestALinkStartsOnTheCallersAccountAndASignInOnNone(t *testing.T) {
	store := inmemory_account_store.New()
	lifetime := accounts.Lifetime{}.WithDefaults()
	require.NoError(t, store.CreateGuest(t.Context(), accounts.GuestSession(accounts.AccountID{15: 7}, accounts.TokenOf("guest"), lifetime, now)))

	linked, err := signin.LinkTarget(t.Context(), store, accounts.IntentLink, "cp_sid=guest", now)
	require.NoError(t, err)
	assert.Equal(t, accounts.AccountID{15: 7}, linked)

	signedIn, err := signin.LinkTarget(t.Context(), store, accounts.IntentSignIn, "cp_sid=guest", now)
	require.NoError(t, err)
	assert.Equal(t, accounts.AccountID{}, signedIn)

	_, err = signin.LinkTarget(t.Context(), store, accounts.IntentLink, "", now)
	assert.ErrorIs(t, err, accounts.ErrNoAccount)
}

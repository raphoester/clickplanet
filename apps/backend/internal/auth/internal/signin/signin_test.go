package signin_test

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
)

var now = time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)

type failingSecrets struct{}

func (failingSecrets) NewSecret() (string, error) { return "", errors.New("no entropy") }

func flow(t *testing.T) *signin.Flow {
	t.Helper()

	flow, err := signin.NewFlow(signin.Google, &signin.SequentialSecrets{}, now)
	require.NoError(t, err)
	return flow
}

func TestAFlowDrawsEachSecretApartAndLastsTenMinutes(t *testing.T) {
	assert.Equal(t, &signin.Flow{
		Provider: "google", State: "secret-1", Verifier: "secret-2", Nonce: "secret-3", ExpiresAt: now.Add(10 * time.Minute),
	}, flow(t))
}

func TestAFlowWithoutEntropyIsNotStarted(t *testing.T) {
	flow, err := signin.NewFlow(signin.Google, failingSecrets{}, now)

	require.Error(t, err)
	assert.Nil(t, flow)
}

func TestTheChallengeIsTheS256OfTheVerifier(t *testing.T) {
	sum := sha256.Sum256([]byte("secret-2"))

	assert.Equal(t, base64.RawURLEncoding.EncodeToString(sum[:]), flow(t).Challenge())
}

func TestAFlowAcceptsItsOwnStateUntilItLapses(t *testing.T) {
	flow := flow(t)

	require.NoError(t, flow.CallbackError("secret-1", now.Add(10*time.Minute-time.Second)))
	require.ErrorIs(t, flow.CallbackError("secret-1", now.Add(10*time.Minute)), signin.ErrFlowInvalid)
	assert.ErrorIs(t, flow.CallbackError("another-state", now), signin.ErrFlowInvalid)
}

func TestTheFlowCookieLivesAsLongAsTheFlowOnThisSiteOnly(t *testing.T) {
	cookie, err := http.ParseSetCookie(flow(t).Cookie("sealed", now))
	require.NoError(t, err)

	assert.Equal(t, "cp_oauth", cookie.Name)
	assert.Equal(t, "sealed", cookie.Value)
	assert.Equal(t, 600, cookie.MaxAge)
	assert.True(t, cookie.HttpOnly)
	assert.True(t, cookie.Secure)
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
	assert.Empty(t, cookie.Domain)

	cleared, err := http.ParseSetCookie(signin.ExpiredFlowCookie())
	require.NoError(t, err)
	assert.Equal(t, "cp_oauth", cleared.Name)
	assert.Negative(t, cleared.MaxAge)
}

func TestNoProviderIsSignInOff(t *testing.T) {
	_, err := signin.Providers{}.Provider(signin.Google)

	assert.ErrorIs(t, err, signin.ErrSignInOff)
}

func TestAProviderNotOfferedIsUnknown(t *testing.T) {
	providers := signin.Providers{signin.Discord: signin.NewFakeProvider(signin.Discord)}

	_, err := providers.Provider(signin.Google)

	require.ErrorIs(t, err, signin.ErrUnknownProvider)
	assert.Equal(t, []string{"discord"}, providers.Names())
}

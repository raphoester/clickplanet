package resume_session_usecase_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/inmemory_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/resume_session_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const ip = "203.0.113.7"

var (
	start    = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	lifetime = accounts.Lifetime{}.WithDefaults()
	ada      = accounts.AccountID{15: 1}
)

type fixture struct {
	sessions *inmemory_account_store.Store
	verifier *cpsession.Verifier
	clock    *cptime.FixedClock
	useCase  *resume_session_usecase.UseCase
}

func setUp(t *testing.T) fixture {
	t.Helper()

	secret, public := cpsession.TestKeyPair()
	signer, err := cpsession.NewSigner(cpsession.SignerConfig{Secret: secret, TTL: time.Hour})
	require.NoError(t, err)
	verifier, err := cpsession.NewVerifier(public)
	require.NoError(t, err)

	sessions := inmemory_account_store.New()
	require.NoError(t, sessions.CreateGuest(t.Context(), accounts.GuestSession(ada, accounts.TokenOf("token-1"), lifetime, start)))

	clock := cptime.NewFixedClock(start)
	return fixture{
		sessions: sessions,
		verifier: verifier,
		clock:    clock,
		useCase:  resume_session_usecase.New(accounts.NewResumer(sessions, lifetime), signer, clock),
	}
}

func (f fixture) resume(t *testing.T, cookieHeader string) *resume_session_usecase.Out {
	t.Helper()

	out, err := f.useCase.Execute(t.Context(), resume_session_usecase.In{IP: ip, CookieHeader: cookieHeader})
	require.NoError(t, err)
	return out
}

func TestTheCookiesSessionComesBackAsATokenThatNamesItsAccountAndProvesNoCheck(t *testing.T) {
	f := setUp(t)

	out := f.resume(t, "cp_sid=token-1")

	require.NotNil(t, out.Token)
	claims, err := f.verifier.Verify(out.Token.Value, ip, start)
	require.NoError(t, err)
	assert.Equal(t, ada, claims.Account)
	assert.False(t, claims.Attested, "only CreateSession, after a Turnstile check, mints the click token")
}

func TestASessionIsExtendedWhenDue(t *testing.T) {
	f := setUp(t)

	assert.Empty(t, f.resume(t, "cp_sid=token-1").SetCookie, "not due yet")

	f.clock.Advance(lifetime.ExtendEvery)
	out := f.resume(t, "cp_sid=token-1")

	cookie, err := http.ParseSetCookie(out.SetCookie)
	require.NoError(t, err)
	assert.Equal(t, "cp_sid", cookie.Name)
	assert.Equal(t, "token-1", cookie.Value)
}

func TestACookieWithNoLiveSessionGetsNoTokenAndStartsNoGuest(t *testing.T) {
	for name, header := range map[string]string{
		"no cookie":      "",
		"unknown cookie": "cp_sid=made-up",
	} {
		t.Run(name, func(t *testing.T) {
			f := setUp(t)

			out := f.resume(t, header)

			assert.Nil(t, out.Token)
			assert.Empty(t, out.SetCookie)
		})
	}

	t.Run("expired", func(t *testing.T) {
		f := setUp(t)
		f.clock.Advance(91 * 24 * time.Hour)

		assert.Nil(t, f.resume(t, "cp_sid=token-1").Token)
	})
}

func TestNoAddressIsRefused(t *testing.T) {
	_, err := setUp(t).useCase.Execute(t.Context(), resume_session_usecase.In{CookieHeader: "cp_sid=token-1"})

	assert.ErrorIs(t, err, resume_session_usecase.ErrNoAddress)
}

func TestAStoreFailureIsAnError(t *testing.T) {
	f := setUp(t)
	f.sessions.FailWith(errors.New("postgres is down"))

	_, err := f.useCase.Execute(t.Context(), resume_session_usecase.In{IP: ip, CookieHeader: "cp_sid=token-1"})

	assert.ErrorContains(t, err, "postgres is down")
}

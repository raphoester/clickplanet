package complete_sign_in_usecase_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/inmemory_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/aes_flow_sealer"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/complete_sign_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/start_sign_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	start    = time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	lifetime = accounts.Lifetime{}.WithDefaults()
	claim    = accounts.Claim{Subject: "google-user", Email: "a@example.com", EmailVerified: true}
)

type fixture struct {
	sealer *aes_flow_sealer.Sealer
	google *signin.FakeProvider
	store  *inmemory_account_store.Store
	clock  *cptime.FixedClock
	start  *start_sign_in_usecase.UseCase
	finish *complete_sign_in_usecase.UseCase
}

func setUp(t *testing.T) *fixture {
	t.Helper()

	sealer, err := aes_flow_sealer.New(bytes.Repeat([]byte{1}, 32))
	require.NoError(t, err)
	f := &fixture{sealer: sealer, google: signin.NewFakeProvider(signin.Google), store: inmemory_account_store.New(), clock: cptime.NewFixedClock(start)}
	providers := signin.Providers{signin.Google: f.google}
	f.start = start_sign_in_usecase.New(providers, &signin.SequentialSecrets{}, sealer, f.clock)
	f.finish = complete_sign_in_usecase.New(providers, sealer, f.store, &accounts.SequentialIDs{}, &accounts.SequentialTokens{}, lifetime, f.clock)
	return f
}

func (f *fixture) guest(t *testing.T, account byte, token string) {
	t.Helper()

	require.NoError(t, f.store.CreateGuest(t.Context(), accounts.GuestSession(accounts.AccountID{15: account}, accounts.TokenOf(token), lifetime, start)))
}

// began starts a sign-in and answers the flow cookie and the state the provider would send back.
func (f *fixture) began(t *testing.T) (string, string) {
	t.Helper()

	out, err := f.start.Execute(t.Context(), signin.Google)
	require.NoError(t, err)
	cookie, err := http.ParseSetCookie(out.SetCookie)
	require.NoError(t, err)
	authorization, err := url.Parse(out.AuthorizationURL)
	require.NoError(t, err)
	return cookie.Name + "=" + cookie.Value, authorization.Query().Get("state")
}

func (f *fixture) complete(t *testing.T, sessionCookie string) (*complete_sign_in_usecase.Out, error) {
	t.Helper()

	flowCookie, state := f.began(t)
	f.google.Grant("the-code", claim)
	cookies := flowCookie
	if sessionCookie != "" {
		cookies += "; " + sessionCookie
	}
	return f.finish.Execute(t.Context(), complete_sign_in_usecase.In{Code: "the-code", State: state, CookieHeader: cookies})
}

func cookieValue(t *testing.T, setCookie string) string {
	t.Helper()

	cookie, err := http.ParseSetCookie(setCookie)
	require.NoError(t, err)
	return cookie.Value
}

func TestANewIdentityLinksToTheGuestAndReplacesItsSession(t *testing.T) {
	f := setUp(t)
	f.guest(t, 7, "guest-token")

	out, err := f.complete(t, "cp_sid=guest-token")
	require.NoError(t, err)

	assert.Equal(t, accounts.AccountID{15: 7}, out.Account)
	assert.Equal(t, accounts.Linked, out.Outcome)
	assert.Equal(t, "token-1", cookieValue(t, out.SetCookie))
	session, err := f.store.Session(t.Context(), accounts.TokenOf("token-1").Hash)
	require.NoError(t, err)
	assert.Equal(t, start.Add(30*24*time.Hour), session.ExpiresAt, "a linked session lasts the linked lifetime")
	_, err = f.store.Session(t.Context(), accounts.TokenOf("guest-token").Hash)
	require.ErrorIs(t, err, accounts.ErrSessionNotFound)
	identity, err := f.store.Identity(t.Context(), signin.Google, "google-user")
	require.NoError(t, err)
	assert.Equal(t, &accounts.Identity{
		Provider: "google", Subject: "google-user", Account: accounts.AccountID{15: 7}, Email: "a@example.com", EmailVerified: true, LinkedAt: start,
	}, identity)
}

func TestAKnownIdentitySignsInToItsAccountAndLeavesTheGuestAsItWas(t *testing.T) {
	f := setUp(t)
	_, err := f.complete(t, "")
	require.NoError(t, err)
	f.guest(t, 7, "guest-token")

	out, err := f.complete(t, "cp_sid=guest-token")
	require.NoError(t, err)

	assert.Equal(t, accounts.AccountID{15: 1}, out.Account)
	assert.Equal(t, accounts.SignedIn, out.Outcome)
	guest, err := f.store.Account(t.Context(), accounts.AccountID{15: 7})
	require.NoError(t, err)
	assert.False(t, guest.Linked(), "nothing is merged, and the guest gains no identity")
}

func TestABrowserWithNoAccountGetsANewOne(t *testing.T) {
	f := setUp(t)

	out, err := f.complete(t, "cp_sid=made-up")
	require.NoError(t, err)

	assert.Equal(t, accounts.AccountID{15: 1}, out.Account)
	assert.Equal(t, accounts.Created, out.Outcome)
}

func TestAnAccountHoldingTheProviderAlreadyIsNotGivenASecondUserOfIt(t *testing.T) {
	f := setUp(t)
	identity := accounts.NewIdentity(signin.Google, accounts.Claim{Subject: "someone-else"}, accounts.AccountID{15: 7}, start)
	require.NoError(t, f.store.SaveSignIn(t.Context(), accounts.SignIn{
		NewAccount: true, Identity: identity, Session: accounts.LinkedSession(identity.Account, accounts.TokenOf("linked"), lifetime, start),
	}))

	out, err := f.complete(t, "cp_sid=linked")
	require.NoError(t, err)

	assert.Equal(t, accounts.Created, out.Outcome)
	assert.NotEqual(t, accounts.AccountID{15: 7}, out.Account)
}

func TestAMatchingEmailLinksNothing(t *testing.T) {
	f := setUp(t)
	identity := accounts.NewIdentity("discord", accounts.Claim{Subject: "discord-user", Email: claim.Email, EmailVerified: true}, accounts.AccountID{15: 7}, start)
	require.NoError(t, f.store.SaveSignIn(t.Context(), accounts.SignIn{
		NewAccount: true, Identity: identity, Session: accounts.LinkedSession(identity.Account, accounts.TokenOf("discord"), lifetime, start),
	}))

	out, err := f.complete(t, "")
	require.NoError(t, err)

	assert.Equal(t, accounts.Created, out.Outcome)
	assert.NotEqual(t, accounts.AccountID{15: 7}, out.Account)
}

func TestACallbackThatDoesNotMatchTheFlowIsRefusedBeforeTheProviderIsAsked(t *testing.T) {
	f := setUp(t)
	flowCookie, state := f.began(t)
	f.google.Grant("the-code", claim)

	for name, in := range map[string]complete_sign_in_usecase.In{
		"no flow cookie": {Code: "the-code", State: state},
		"another state":  {Code: "the-code", State: "forged", CookieHeader: flowCookie},
		"forged cookie":  {Code: "the-code", State: state, CookieHeader: "cp_oauth=forged"},
	} {
		t.Run(name, func(t *testing.T) {
			out, err := f.finish.Execute(t.Context(), in)

			require.ErrorIs(t, err, signin.ErrFlowInvalid)
			assert.Nil(t, out)
		})
	}

	f.clock.Advance(signin.FlowTTL)
	_, err := f.finish.Execute(t.Context(), complete_sign_in_usecase.In{Code: "the-code", State: state, CookieHeader: flowCookie})
	require.ErrorIs(t, err, signin.ErrFlowInvalid, "a lapsed flow")

	f.clock.Advance(-signin.FlowTTL)
	_, err = f.finish.Execute(t.Context(), complete_sign_in_usecase.In{Code: "the-code", State: state, CookieHeader: flowCookie})
	assert.NoError(t, err, "the code was never spent on a refused callback")
}

func TestACodeTheProviderRefusesSignsNothingIn(t *testing.T) {
	f := setUp(t)
	flowCookie, state := f.began(t)

	out, err := f.finish.Execute(t.Context(), complete_sign_in_usecase.In{Code: "never-granted", State: state, CookieHeader: flowCookie})

	require.ErrorIs(t, err, signin.ErrProviderRefused)
	assert.Nil(t, out)
}

func TestAStoreFailureFailsTheSignIn(t *testing.T) {
	f := setUp(t)
	flowCookie, state := f.began(t)
	f.google.Grant("the-code", claim)
	f.store.FailWith(errors.New("postgres is down"))

	_, err := f.finish.Execute(t.Context(), complete_sign_in_usecase.In{Code: "the-code", State: state, CookieHeader: flowCookie + "; cp_sid=guest"})

	require.Error(t, err)
	assert.NotErrorIs(t, err, signin.ErrFlowInvalid)
	assert.NotErrorIs(t, err, signin.ErrProviderRefused)
}

func TestSignInOffCompletesNothing(t *testing.T) {
	f := setUp(t)
	useCase := complete_sign_in_usecase.New(signin.Providers{}, f.sealer, f.store, &accounts.SequentialIDs{}, &accounts.SequentialTokens{}, lifetime, f.clock)

	_, err := useCase.Execute(t.Context(), complete_sign_in_usecase.In{Code: "the-code", State: "state"})

	assert.ErrorIs(t, err, signin.ErrSignInOff)
}

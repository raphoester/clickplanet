package start_sign_in_usecase_test

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/inmemory_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/aes_flow_sealer"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/start_sign_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)

func setUp(t *testing.T, providers signin.Providers) (*start_sign_in_usecase.UseCase, *aes_flow_sealer.Sealer) {
	t.Helper()

	sealer, err := aes_flow_sealer.New(bytes.Repeat([]byte{1}, 32))
	require.NoError(t, err)
	store := inmemory_account_store.New()
	lifetime := accounts.Lifetime{}.WithDefaults()
	require.NoError(t, store.CreateGuest(t.Context(), accounts.GuestSession(accounts.AccountID{15: 7}, accounts.TokenOf("guest-token"), lifetime, now)))
	return start_sign_in_usecase.New(providers, store, &signin.SequentialSecrets{}, sealer, cptime.NewFixedClock(now)), sealer
}

func openedFlow(t *testing.T, sealer *aes_flow_sealer.Sealer, out *start_sign_in_usecase.Out) *signin.Flow {
	t.Helper()

	cookie, err := http.ParseSetCookie(out.SetCookie)
	require.NoError(t, err)
	flow, err := sealer.Opened(cookie.Value)
	require.NoError(t, err)
	return flow
}

func TestAStartedSignInSendsTheBrowserToTheProviderAndSealsTheFlowInItsCookie(t *testing.T) {
	useCase, sealer := setUp(t, signin.Providers{signin.Google: signin.NewFakeProvider(signin.Google)})

	out, err := useCase.Execute(t.Context(), start_sign_in_usecase.In{Provider: signin.Google, CookieHeader: "cp_sid=guest-token"})
	require.NoError(t, err)

	assert.Equal(t, "https://google.example/authorize?state=secret-1", out.AuthorizationURL)
	assert.Equal(t, &signin.Flow{
		Provider: "google", State: "secret-1", Verifier: "secret-2", Nonce: "secret-3", ExpiresAt: now.Add(signin.FlowTTL),
	}, openedFlow(t, sealer, out), "a sign-in holds no account: it goes wherever the identity is")
}

func TestALinkSealsTheAccountItStartedOn(t *testing.T) {
	useCase, sealer := setUp(t, signin.Providers{signin.Google: signin.NewFakeProvider(signin.Google)})

	out, err := useCase.Execute(t.Context(), start_sign_in_usecase.In{Provider: signin.Google, Intent: accounts.IntentLink, CookieHeader: "cp_sid=guest-token"})
	require.NoError(t, err)

	flow := openedFlow(t, sealer, out)
	assert.Equal(t, accounts.IntentLink, flow.Intent)
	assert.Equal(t, accounts.AccountID{15: 7}, flow.Account)
}

func TestALinkFromABrowserWithNoAccountStartsNothing(t *testing.T) {
	useCase, _ := setUp(t, signin.Providers{signin.Google: signin.NewFakeProvider(signin.Google)})

	for name, cookie := range map[string]string{"no cookie": "", "unknown session": "cp_sid=made-up"} {
		t.Run(name, func(t *testing.T) {
			out, err := useCase.Execute(t.Context(), start_sign_in_usecase.In{Provider: signin.Google, Intent: accounts.IntentLink, CookieHeader: cookie})

			require.ErrorIs(t, err, accounts.ErrNoAccount)
			assert.Nil(t, out)
		})
	}
}

func TestAProviderThatIsNotOfferedStartsNothing(t *testing.T) {
	for name, providers := range map[string]signin.Providers{
		"sign-in off":      {},
		"another provider": {signin.Discord: signin.NewFakeProvider(signin.Discord)},
	} {
		t.Run(name, func(t *testing.T) {
			useCase, _ := setUp(t, providers)

			out, err := useCase.Execute(t.Context(), start_sign_in_usecase.In{Provider: signin.Google})

			require.Error(t, err)
			assert.Nil(t, out)
		})
	}
}

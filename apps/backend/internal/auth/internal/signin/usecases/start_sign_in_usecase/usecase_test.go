package start_sign_in_usecase_test

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	return start_sign_in_usecase.New(providers, &signin.SequentialSecrets{}, sealer, cptime.NewFixedClock(now)), sealer
}

func TestAStartedSignInSendsTheBrowserToTheProviderAndSealsTheFlowInItsCookie(t *testing.T) {
	useCase, sealer := setUp(t, signin.Providers{signin.Google: signin.NewFakeProvider(signin.Google)})

	out, err := useCase.Execute(t.Context(), signin.Google)
	require.NoError(t, err)

	assert.Equal(t, "https://google.example/authorize?state=secret-1", out.AuthorizationURL)
	cookie, err := http.ParseSetCookie(out.SetCookie)
	require.NoError(t, err)
	flow, err := sealer.Opened(cookie.Value)
	require.NoError(t, err)
	assert.Equal(t, &signin.Flow{Provider: "google", State: "secret-1", Verifier: "secret-2", Nonce: "secret-3", ExpiresAt: now.Add(signin.FlowTTL)}, flow)
}

func TestAProviderThatIsNotOfferedStartsNothing(t *testing.T) {
	for name, providers := range map[string]signin.Providers{
		"sign-in off":      {},
		"another provider": {signin.Discord: signin.NewFakeProvider(signin.Discord)},
	} {
		t.Run(name, func(t *testing.T) {
			useCase, _ := setUp(t, providers)

			out, err := useCase.Execute(t.Context(), signin.Google)

			require.Error(t, err)
			assert.Nil(t, out)
		})
	}
}

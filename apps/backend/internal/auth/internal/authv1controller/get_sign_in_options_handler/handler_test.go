package get_sign_in_options_handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_sign_in_options_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cphttpserver"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

func TestEveryOfferedProviderIsAnswered(t *testing.T) {
	offer := signin.Providers{
		signin.Google:  signin.NewFakeProvider(signin.Google),
		signin.Discord: signin.NewFakeProvider(signin.Discord),
	}

	res, err := get_sign_in_options_handler.New(offer).GetSignInOptions(t.Context(), connect.NewRequest(&authv1.GetSignInOptionsRequest{}))
	require.NoError(t, err)

	assert.Equal(t, []authv1.Provider{authv1.Provider_PROVIDER_DISCORD, authv1.Provider_PROVIDER_GOOGLE}, res.Msg.GetProviders())
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestSignInOffIsAnEmptyListAndNotAnError(t *testing.T) {
	res, err := get_sign_in_options_handler.New(signin.Providers{}).GetSignInOptions(t.Context(), connect.NewRequest(&authv1.GetSignInOptionsRequest{}))
	require.NoError(t, err)

	assert.Empty(t, res.Msg.GetProviders())
}

// onlyGetSignInOptions serves GetSignInOptions, and Unimplemented for every other procedure.
type onlyGetSignInOptions struct {
	unimplemented
	get_sign_in_options_handler.GetSignInOptionsHandler
}

type unimplemented struct {
	authv1connect.UnimplementedAuthServiceHandler
}

type refuseAll struct{}

func (refuseAll) Take(string) (bool, cpratelimit.State) { return false, cpratelimit.State{} }

// A client asks on every page load, so the question must not spend the mint budget the sign-in itself needs.
func TestAskingIsNotThrottled(t *testing.T) {
	service := onlyGetSignInOptions{GetSignInOptionsHandler: get_sign_in_options_handler.New(signin.Providers{})}
	mux := http.NewServeMux()
	mux.Handle(authv1connect.NewAuthServiceHandler(service, connect.WithInterceptors(authv1controller.NewRateLimitInterceptor(refuseAll{}))))
	httpServer := httptest.NewServer(cphttpserver.IPReaderMiddleware(mux))
	t.Cleanup(httpServer.Close)

	req := connect.NewRequest(&authv1.GetSignInOptionsRequest{})
	req.Header().Set("X-Real-IP", "203.0.113.7")
	_, err := authv1connect.NewAuthServiceClient(httpServer.Client(), httpServer.URL).GetSignInOptions(t.Context(), req)

	assert.NoError(t, err)
}

package cpcallers_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcallers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

var ada = cpsession.AccountID{15: 1}

type stubAuth struct {
	authv1connect.UnimplementedInternalServiceHandler

	accounts map[string]string
	err      error
}

func (s stubAuth) GetCaller(
	_ context.Context,
	req *connect.Request[authv1.GetCallerRequest],
) (*connect.Response[authv1.GetCallerResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&authv1.GetCallerResponse{AccountId: s.accounts[req.Msg.GetCookie()]}), nil
}

func callers(t *testing.T, auth stubAuth) *cpcallers.Callers {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(authv1connect.NewInternalServiceHandler(auth))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return cpcallers.New(authv1connect.NewInternalServiceClient(server.Client(), server.URL))
}

func TestItAnswersTheAccountTheCookieNames(t *testing.T) {
	auth := stubAuth{accounts: map[string]string{"cp_sid=token-1": ada.String()}}

	account, err := callers(t, auth).Caller(t.Context(), "cp_sid=token-1")

	require.NoError(t, err)
	assert.Equal(t, ada, account)
}

func TestACookieThatNamesNobodyIsNoAccount(t *testing.T) {
	account, err := callers(t, stubAuth{}).Caller(t.Context(), "cp_sid=made-up")

	require.NoError(t, err)
	assert.Equal(t, cpsession.NoAccount, account)
}

func TestAnAuthModuleThatFailsIsAnError(t *testing.T) {
	auth := stubAuth{err: connect.NewError(connect.CodeInternal, errors.New("postgres is down"))}

	_, err := callers(t, auth).Caller(t.Context(), "cp_sid=token-1")

	assert.ErrorContains(t, err, "failed to ask the auth module")
}

func TestAnAuthModuleThatIsNotListeningIsAnError(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()

	_, err := cpcallers.New(authv1connect.NewInternalServiceClient(server.Client(), server.URL)).
		Caller(t.Context(), "cp_sid=token-1")

	assert.ErrorContains(t, err, "failed to ask the auth module")
}

func logged(inner *cpcallers.Callers) (*cpcallers.Logged, *bytes.Buffer) {
	var out bytes.Buffer
	return cpcallers.NewLogged(inner, slog.New(slog.NewTextHandler(&out, nil))), &out
}

func TestAFailureIsLoggedWithoutTheCookieAndPassedOn(t *testing.T) {
	failing := callers(t, stubAuth{err: connect.NewError(connect.CodeInternal, errors.New("auth is stuck"))})
	caller, out := logged(failing)

	_, err := caller.Caller(t.Context(), "cp_sid=secret-token")

	require.ErrorContains(t, err, "auth is stuck")
	assert.Contains(t, out.String(), "level=ERROR")
	assert.Contains(t, out.String(), "auth is stuck")
	assert.NotContains(t, out.String(), "secret-token", "a cookie signs a browser in")
}

func TestAnAnswerIsPassedOnAndNotLogged(t *testing.T) {
	caller, out := logged(callers(t, stubAuth{accounts: map[string]string{"cp_sid=token-1": ada.String()}}))

	account, err := caller.Caller(t.Context(), "cp_sid=token-1")

	require.NoError(t, err)
	assert.Equal(t, ada, account)
	assert.Empty(t, out.String())
}

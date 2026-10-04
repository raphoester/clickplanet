package rpc_auth_callers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/rpc_auth_callers"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

var ada = messages.AccountID{15: 1}

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

type dialer struct {
	client connect.HTTPClient
	url    string
	err    error
}

func (d dialer) Dial() (connect.HTTPClient, string, error) {
	return d.client, d.url, d.err
}

func callers(t *testing.T, auth stubAuth) *rpc_auth_callers.Callers {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(authv1connect.NewInternalServiceHandler(auth))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_auth_callers.New(dialer{client: server.Client(), url: server.URL})
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
	assert.Equal(t, messages.NoAccount, account)
}

func TestAnAuthModuleThatFailsIsAnError(t *testing.T) {
	auth := stubAuth{err: connect.NewError(connect.CodeInternal, errors.New("postgres is down"))}

	_, err := callers(t, auth).Caller(t.Context(), "cp_sid=token-1")

	assert.ErrorContains(t, err, "failed to ask the auth module")
}

func TestAnUnreachableAuthModuleIsAnError(t *testing.T) {
	_, err := rpc_auth_callers.New(dialer{err: errors.New("no internal listener")}).Caller(t.Context(), "cp_sid=token-1")

	assert.ErrorContains(t, err, "failed to reach the auth module")
}

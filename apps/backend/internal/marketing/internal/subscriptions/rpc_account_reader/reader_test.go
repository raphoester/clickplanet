package rpc_account_reader_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/rpc_account_reader"
)

type stubAuth struct {
	authv1connect.UnimplementedInternalServiceHandler

	answers map[string]*authv1.GetAccountResponse
	err     error
}

func (s stubAuth) GetAccount(
	_ context.Context,
	req *connect.Request[authv1.GetAccountRequest],
) (*connect.Response[authv1.GetAccountResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	if answer, found := s.answers[req.Msg.GetAccountId()]; found {
		return connect.NewResponse(answer), nil
	}
	return connect.NewResponse(&authv1.GetAccountResponse{}), nil
}

type dialer struct {
	client connect.HTTPClient
	url    string
	err    error
}

func (d dialer) Dial() (connect.HTTPClient, string, error) {
	return d.client, d.url, d.err
}

var ada = subscriptions.AccountID{15: 1}

func reader(t *testing.T, auth stubAuth) *rpc_account_reader.Reader {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(authv1connect.NewInternalServiceHandler(auth))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_account_reader.New(dialer{client: server.Client(), url: server.URL})
}

func TestTheAccountIsWhatAuthSaysWithItsAddressesInOrder(t *testing.T) {
	accounts := reader(t, stubAuth{answers: map[string]*authv1.GetAccountResponse{
		ada.String(): {Linked: true, Emails: []string{"Ada@Example.com", "ada@gmail.com"}},
	}})

	account, err := accounts.Account(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, subscriptions.Account{
		ID: ada, Linked: true, Addresses: []subscriptions.Address{"ada@example.com", "ada@gmail.com"},
	}, account)
}

func TestAnAddressThisModuleCannotMailIsLeftOut(t *testing.T) {
	accounts := reader(t, stubAuth{answers: map[string]*authv1.GetAccountResponse{
		ada.String(): {Linked: true, Emails: []string{"ada@localhost", "ada@example.com"}},
	}})

	account, err := accounts.Account(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, []subscriptions.Address{"ada@example.com"}, account.Addresses)
}

func TestAnAccountAuthDoesNotKnowIsAGuestWithNoAddress(t *testing.T) {
	account, err := reader(t, stubAuth{}).Account(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, subscriptions.Account{ID: ada, Addresses: []subscriptions.Address{}}, account)
}

func TestAnAuthThatFailsIsAnErrorAndNotAGuest(t *testing.T) {
	_, err := reader(t, stubAuth{err: connect.NewError(connect.CodeUnavailable, errors.New("auth is down"))}).
		Account(t.Context(), ada)

	assert.ErrorContains(t, err, "failed to ask auth")
}

func TestAnUnreachableAuthIsAnError(t *testing.T) {
	_, err := rpc_account_reader.New(dialer{err: errors.New("no internal listener")}).Account(t.Context(), ada)

	assert.ErrorContains(t, err, "failed to reach the auth module")
}

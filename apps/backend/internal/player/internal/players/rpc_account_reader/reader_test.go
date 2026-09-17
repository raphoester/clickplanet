package rpc_account_reader_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/rpc_account_reader"
)

// stubAuth answers what the auth module answers for the accounts it was told are linked.
type stubAuth struct {
	authv1connect.UnimplementedInternalServiceHandler

	linked  map[string]bool
	created map[string]time.Time
	err     error
}

func (s stubAuth) GetAccount(
	_ context.Context,
	req *connect.Request[authv1.GetAccountRequest],
) (*connect.Response[authv1.GetAccountResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	res := &authv1.GetAccountResponse{Linked: s.linked[req.Msg.GetAccountId()]}
	if created, ok := s.created[req.Msg.GetAccountId()]; ok {
		res.CreatedAtUnixMs = created.UnixMilli()
	}
	return connect.NewResponse(res), nil
}

type dialer struct {
	client connect.HTTPClient
	url    string
	err    error
}

func (d dialer) Dial() (connect.HTTPClient, string, error) {
	return d.client, d.url, d.err
}

var (
	ada   = players.AccountID{15: 1}
	guest = players.AccountID{15: 2}
)

func reader(t *testing.T, auth stubAuth) *rpc_account_reader.Reader {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(authv1connect.NewInternalServiceHandler(auth))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_account_reader.New(dialer{client: server.Client(), url: server.URL})
}

func TestItAnswersWhatAuthSays(t *testing.T) {
	accounts := reader(t, stubAuth{linked: map[string]bool{ada.String(): true}})

	linked, err := accounts.Linked(t.Context(), ada)
	require.NoError(t, err)
	assert.True(t, linked)

	linked, err = accounts.Linked(t.Context(), guest)
	require.NoError(t, err)
	assert.False(t, linked)
}

func TestItSaysWhenAuthMadeTheAccount(t *testing.T) {
	createdAt := time.Date(2026, 9, 1, 8, 30, 0, 0, time.UTC)
	accounts := reader(t, stubAuth{created: map[string]time.Time{ada.String(): createdAt}})

	at, err := accounts.CreatedAt(t.Context(), ada)
	require.NoError(t, err)
	assert.Equal(t, createdAt, at)

	at, err = accounts.CreatedAt(t.Context(), guest)
	require.NoError(t, err)
	assert.True(t, at.IsZero(), "an account auth does not know")
}

func TestAnAuthThatFailsIsAnErrorAndNotAGuest(t *testing.T) {
	_, err := reader(t, stubAuth{err: connect.NewError(connect.CodeNotFound, errors.New("auth is off"))}).
		Linked(t.Context(), ada)

	assert.ErrorContains(t, err, "failed to ask auth")

	_, err = reader(t, stubAuth{err: connect.NewError(connect.CodeNotFound, errors.New("auth is off"))}).
		CreatedAt(t.Context(), ada)

	assert.ErrorContains(t, err, "failed to ask auth")
}

func TestAnUnreachableAuthIsAnError(t *testing.T) {
	_, err := rpc_account_reader.New(dialer{err: errors.New("no internal listener")}).Linked(t.Context(), ada)

	assert.ErrorContains(t, err, "failed to reach the auth module")
}

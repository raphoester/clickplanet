package rpc_player_accounts_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports/rpc_player_accounts"
)

type stubPlayer struct {
	playerv1connect.UnimplementedInternalServiceHandler

	accounts map[string]string
	err      error
}

func (s stubPlayer) FindAccount(
	_ context.Context,
	req *connect.Request[playerv1.FindAccountRequest],
) (*connect.Response[playerv1.FindAccountResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	account, found := s.accounts[req.Msg.GetName()]
	if !found {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("no account is shown under this name"))
	}
	return connect.NewResponse(&playerv1.FindAccountResponse{AccountId: account}), nil
}

var ada = reports.AccountID{15: 1}

func accounts(t *testing.T, player stubPlayer) *rpc_player_accounts.Accounts {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(playerv1connect.NewInternalServiceHandler(player))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_player_accounts.New(playerv1connect.NewInternalServiceClient(server.Client(), server.URL))
}

func TestANameIsAnsweredWithTheAccountThePlayerModuleShowsItFor(t *testing.T) {
	account, err := accounts(t, stubPlayer{accounts: map[string]string{"Ada_L": ada.String()}}).AccountOf(t.Context(), "Ada_L")

	require.NoError(t, err)
	assert.Equal(t, ada, account)
}

func TestANameThePlayerModuleShowsForNobodyIsNoPlayer(t *testing.T) {
	_, err := accounts(t, stubPlayer{}).AccountOf(t.Context(), "Bob")

	require.ErrorIs(t, err, reports.ErrNoPlayer)
}

func TestAFailureToAskIsNotNoPlayer(t *testing.T) {
	_, err := accounts(t, stubPlayer{err: connect.NewError(connect.CodeInternal, errors.New("postgres is down"))}).
		AccountOf(t.Context(), "Ada_L")

	require.Error(t, err)
	assert.NotErrorIs(t, err, reports.ErrNoPlayer)
}

func TestAnAnswerThatIsNotAnAccountIsAnError(t *testing.T) {
	_, err := accounts(t, stubPlayer{accounts: map[string]string{"Ada_L": "not-an-account"}}).AccountOf(t.Context(), "Ada_L")

	require.Error(t, err)
	assert.NotErrorIs(t, err, reports.ErrNoPlayer)
}

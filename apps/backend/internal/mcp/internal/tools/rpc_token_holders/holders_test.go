package rpc_token_holders_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/mcp/internal/tools"
	"github.com/raphoester/clickplanet.lol-backend/internal/mcp/internal/tools/rpc_token_holders"
)

type auth map[string]error

func (a auth) GetTokenOperator(
	_ context.Context,
	req *connect.Request[authv1.GetTokenOperatorRequest],
) (*connect.Response[authv1.GetTokenOperatorResponse], error) {
	if err, refused := a[req.Msg.GetToken()]; refused {
		return nil, err
	}
	return connect.NewResponse(&authv1.GetTokenOperatorResponse{AccountId: "the account", Client: "Claude"}), nil
}

func TestAuthsAnswerIsWhetherTheTokenCallsOrWhyNot(t *testing.T) {
	holders := rpc_token_holders.New(auth{
		"cpat_lapsed": connect.NewError(connect.CodeUnauthenticated, errors.New("lapsed")),
		"cpat_bob":    connect.NewError(connect.CodePermissionDenied, errors.New("a player")),
		"cpat_down":   connect.NewError(connect.CodeUnavailable, errors.New("auth is down")),
	})

	require.NoError(t, holders.HolderError(t.Context(), "cpat_ada"))
	require.ErrorIs(t, holders.HolderError(t.Context(), "cpat_lapsed"), tools.ErrNoToken)
	require.ErrorIs(t, holders.HolderError(t.Context(), "cpat_bob"), tools.ErrNotAnOperator)

	err := holders.HolderError(t.Context(), "cpat_down")
	require.Error(t, err)
	assert.NotErrorIs(t, err, tools.ErrNoToken, "auth down is not a refusal")
}

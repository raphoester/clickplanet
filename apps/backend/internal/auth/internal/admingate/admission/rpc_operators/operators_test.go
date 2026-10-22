package rpc_operators_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admingate/admission"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admingate/admission/rpc_operators"
)

type auth map[string]error

func (a auth) GetOperator(
	_ context.Context,
	req *connect.Request[authv1.GetOperatorRequest],
) (*connect.Response[authv1.GetOperatorResponse], error) {
	if err, refused := a[req.Msg.GetCookie()]; refused {
		return nil, err
	}
	return connect.NewResponse(&authv1.GetOperatorResponse{AccountId: "the account"}), nil
}

func (a auth) GetTokenOperator(
	_ context.Context,
	req *connect.Request[authv1.GetTokenOperatorRequest],
) (*connect.Response[authv1.GetTokenOperatorResponse], error) {
	if err, refused := a[req.Msg.GetToken()]; refused {
		return nil, err
	}
	return connect.NewResponse(&authv1.GetTokenOperatorResponse{AccountId: "the account", Client: "Claude"}), nil
}

func refusals() auth {
	return auth{
		"":              connect.NewError(connect.CodeUnauthenticated, errors.New("no session")),
		"cp_sid=player": connect.NewError(connect.CodePermissionDenied, errors.New("a player")),
		"cp_sid=down":   connect.NewError(connect.CodeUnavailable, errors.New("auth is down")),
		"cpat_unknown":  connect.NewError(connect.CodeUnauthenticated, errors.New("no such token")),
		"cpat_player":   connect.NewError(connect.CodePermissionDenied, errors.New("a player")),
		"cpat_down":     connect.NewError(connect.CodeUnavailable, errors.New("auth is down")),
	}
}

func TestAuthsAnswerIsTheOperatorOrTheReasonItIsNot(t *testing.T) {
	operators := rpc_operators.New(refusals())

	operator, err := operators.Operator(t.Context(), "cp_sid=operator")
	require.NoError(t, err)
	assert.Equal(t, admission.Operator("the account"), operator)

	_, err = operators.Operator(t.Context(), "")
	require.ErrorIs(t, err, admission.ErrNoSession)

	_, err = operators.Operator(t.Context(), "cp_sid=player")
	require.ErrorIs(t, err, admission.ErrNotAnOperator)

	_, err = operators.Operator(t.Context(), "cp_sid=down")
	require.Error(t, err)
	assert.NotErrorIs(t, err, admission.ErrNoSession, "auth down is not a refusal")
	assert.NotErrorIs(t, err, admission.ErrNotAnOperator, "auth down is not a refusal")
}

func TestAuthsAnswerAboutATokenIsItsConnectionOrTheReasonItIsNot(t *testing.T) {
	operators := rpc_operators.New(refusals())

	caller, err := operators.Connection(t.Context(), "cpat_claude")
	require.NoError(t, err)
	assert.Equal(t, admission.ConnectionOf("the account", "Claude"), caller)

	_, err = operators.Connection(t.Context(), "cpat_unknown")
	require.ErrorIs(t, err, admission.ErrNoToken)

	_, err = operators.Connection(t.Context(), "cpat_player")
	require.ErrorIs(t, err, admission.ErrNotAnOperator)

	_, err = operators.Connection(t.Context(), "cpat_down")
	require.Error(t, err)
	assert.NotErrorIs(t, err, admission.ErrNoToken, "auth down is not a refusal")
}

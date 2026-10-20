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

func TestAuthsAnswerIsTheOperatorOrTheReasonItIsNot(t *testing.T) {
	operators := rpc_operators.New(auth{
		"":              connect.NewError(connect.CodeUnauthenticated, errors.New("no session")),
		"cp_sid=player": connect.NewError(connect.CodePermissionDenied, errors.New("a player")),
		"cp_sid=down":   connect.NewError(connect.CodeUnavailable, errors.New("auth is down")),
	})

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

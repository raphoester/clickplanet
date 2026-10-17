package get_operator_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_operator_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_operator_handler/operator_query"
)

type stubQuery map[string]error

func (q stubQuery) Operator(_ context.Context, cookieHeader string) (*authv1.GetOperatorResponse, error) {
	if err, refused := q[cookieHeader]; refused {
		return nil, err
	}
	return &authv1.GetOperatorResponse{AccountId: "the account of " + cookieHeader}, nil
}

func operator(t *testing.T, cookie string) (*connect.Response[authv1.GetOperatorResponse], error) {
	t.Helper()

	handler := get_operator_handler.New(stubQuery{
		"":                fmt.Errorf("%w: no cookie", operator_query.ErrNoSession),
		"cp_sid=player":   fmt.Errorf("%w: a player", operator_query.ErrNotAnOperator),
		"cp_sid=database": errors.New("postgres is down"),
	})
	return handler.GetOperator(t.Context(), connect.NewRequest(&authv1.GetOperatorRequest{Cookie: cookie})) //nolint:wrapcheck // the tests read the code.
}

func TestAnOperatorIsAnsweredWithItsAccount(t *testing.T) {
	res, err := operator(t, "cp_sid=operator")

	require.NoError(t, err)
	assert.Equal(t, "the account of cp_sid=operator", res.Msg.GetAccountId())
}

func TestEachRefusalHasTheCodeOfItsReason(t *testing.T) {
	_, err := operator(t, "")
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	_, err = operator(t, "cp_sid=player")
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))

	_, err = operator(t, "cp_sid=database")
	require.Error(t, err)
	var coded *connect.Error
	assert.NotErrorAs(t, err, &coded, "a failure is the error net's to answer")
}

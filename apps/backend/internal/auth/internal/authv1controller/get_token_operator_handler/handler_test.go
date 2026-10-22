package get_token_operator_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_token_operator_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_token_operator_handler/token_operator_query"
)

type stubQuery map[string]error

func (q stubQuery) Operator(_ context.Context, token string) (*authv1.GetTokenOperatorResponse, error) {
	if err, refused := q[token]; refused {
		return nil, err
	}
	return &authv1.GetTokenOperatorResponse{AccountId: "the account of " + token, Client: "Claude"}, nil
}

func operator(t *testing.T, token string) (*connect.Response[authv1.GetTokenOperatorResponse], error) {
	t.Helper()

	handler := get_token_operator_handler.New(stubQuery{
		"":          fmt.Errorf("%w: no token", token_operator_query.ErrNoToken),
		"cpat_bob":  fmt.Errorf("%w: a player", token_operator_query.ErrNotAnOperator),
		"cpat_down": errors.New("postgres is down"),
	})
	return handler.GetTokenOperator(t.Context(), connect.NewRequest(&authv1.GetTokenOperatorRequest{Token: token})) //nolint:wrapcheck // the tests read the code.
}

func TestATokenIsAnsweredWithItsOperator(t *testing.T) {
	res, err := operator(t, "cpat_ada")

	require.NoError(t, err)
	assert.Equal(t, "the account of cpat_ada", res.Msg.GetAccountId())
}

func TestEachRefusalHasTheCodeOfItsReason(t *testing.T) {
	_, err := operator(t, "")
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	_, err = operator(t, "cpat_bob")
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))

	_, err = operator(t, "cpat_down")
	require.Error(t, err)
	var coded *connect.Error
	assert.NotErrorAs(t, err, &coded, "a failure is the error net's to answer")
}

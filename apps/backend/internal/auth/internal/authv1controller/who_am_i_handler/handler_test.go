package who_am_i_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_operator_handler/operator_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/who_am_i_handler"
)

type stubQuery map[string]error

func (q stubQuery) Operator(_ context.Context, cookieHeader string) (*authv1.GetOperatorResponse, error) {
	if err, refused := q[cookieHeader]; refused {
		return nil, err
	}
	return &authv1.GetOperatorResponse{AccountId: "the account of " + cookieHeader}, nil
}

func whoAmI(t *testing.T, cookie string) (*connect.Response[authv1.WhoAmIResponse], error) {
	t.Helper()

	handler := who_am_i_handler.New(stubQuery{
		"":                fmt.Errorf("%w: no cookie", operator_query.ErrNoSession),
		"cp_sid=player":   fmt.Errorf("%w: a player", operator_query.ErrNotAnOperator),
		"cp_sid=database": errors.New("postgres is down"),
	})
	req := connect.NewRequest(&authv1.WhoAmIRequest{})
	if cookie != "" {
		req.Header().Set("Cookie", cookie)
	}
	return handler.WhoAmI(t.Context(), req) //nolint:wrapcheck // the tests read the code.
}

func TestTheOperatorTheCookieNamesIsTheAnswerAndIsNeverCached(t *testing.T) {
	res, err := whoAmI(t, "cp_sid=operator")

	require.NoError(t, err)
	assert.Equal(t, "the account of cp_sid=operator", res.Msg.GetAccountId())
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestEachRefusalHasTheCodeOfItsReason(t *testing.T) {
	_, err := whoAmI(t, "")
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	_, err = whoAmI(t, "cp_sid=player")
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))

	_, err = whoAmI(t, "cp_sid=database")
	require.Error(t, err)
	var coded *connect.Error
	assert.NotErrorAs(t, err, &coded, "a failure is the error net's to answer")
}

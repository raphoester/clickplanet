package approve_connection_handler_test

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/approve_connection_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_operator_handler/operator_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/usecases/approve_connection_usecase"
)

const ada = "00000000-0000-0000-0000-000000000001"

type operators struct{}

func (operators) Operator(_ context.Context, cookie string) (*authv1.GetOperatorResponse, error) {
	switch cookie {
	case "cp_admin=ada":
		return &authv1.GetOperatorResponse{AccountId: ada}, nil
	case "cp_admin=bob":
		return nil, operator_query.ErrNotAnOperator
	}
	return nil, operator_query.ErrNoSession
}

type useCase struct {
	asked *approve_connection_usecase.In
}

func (u useCase) Execute(_ context.Context, in approve_connection_usecase.In) (approve_connection_usecase.Out, error) {
	*u.asked = in
	if in.Request == "lapsed" {
		return approve_connection_usecase.Out{}, fmt.Errorf("%w: ten minutes ago", connections.ErrExpired)
	}
	return approve_connection_usecase.Out{RedirectURL: "https://claude.ai/cb?code=the+code"}, nil
}

func approve(t *testing.T, cookie, request string) (*connect.Response[authv1.ApproveConnectionResponse], approve_connection_usecase.In, error) {
	t.Helper()
	var asked approve_connection_usecase.In
	req := connect.NewRequest(&authv1.ApproveConnectionRequest{Request: request})
	req.Header().Set("Cookie", cookie)
	res, err := approve_connection_handler.New(operators{}, useCase{asked: &asked}).ApproveConnection(t.Context(), req)
	return res, asked, err //nolint:wrapcheck // the tests read the code.
}

func TestTheAppIsLetInAsTheOperatorOfTheCookie(t *testing.T) {
	res, asked, err := approve(t, "cp_admin=ada", "sealed")

	require.NoError(t, err)
	assert.Equal(t, "https://claude.ai/cb?code=the+code", res.Msg.GetRedirectUrl())
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
	account, err := accounts.AccountIDOf(ada)
	require.NoError(t, err)
	assert.Equal(t, approve_connection_usecase.In{Request: "sealed", Operator: account}, asked)
}

func TestNobodyButAnOperatorLetsAnAppIn(t *testing.T) {
	_, asked, err := approve(t, "", "sealed")
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.Empty(t, asked.Request, "nothing is approved")

	_, _, err = approve(t, "cp_admin=bob", "sealed")
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))

	_, _, err = approve(t, "cp_admin=ada", "lapsed")
	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

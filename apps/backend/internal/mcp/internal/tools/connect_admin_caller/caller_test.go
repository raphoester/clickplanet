package connect_admin_caller_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/mcp/internal/tools"
	"github.com/raphoester/clickplanet.lol-backend/internal/mcp/internal/tools/connect_admin_caller"
)

type admin struct {
	planetv1connect.UnimplementedAdminServiceHandler
	authorization *string
}

func (a admin) BanPlayer(_ context.Context, req *connect.Request[planetv1.BanPlayerRequest]) (*connect.Response[planetv1.BanPlayerResponse], error) {
	*a.authorization = req.Header().Get("Authorization")
	if req.Msg.GetAccountId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("a scope or an account"))
	}
	return connect.NewResponse(&planetv1.BanPlayerResponse{AccountId: req.Msg.GetAccountId(), Offence: 2}), nil
}

func serve(t *testing.T) (*connect_admin_caller.Caller, *string) {
	t.Helper()
	var authorization string
	path, handler := planetv1connect.NewAdminServiceHandler(admin{authorization: &authorization})
	router := http.NewServeMux()
	router.Handle(path, handler)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return connect_admin_caller.New(server.Client(), server.URL), &authorization
}

func TestTheArgumentsAreTheRequestAndTheAnswerIsItsJSON(t *testing.T) {
	caller, authorization := serve(t)

	result, err := caller.Call(t.Context(), planetv1connect.AdminServiceBanPlayerProcedure, "cpat_ada",
		json.RawMessage(`{"accountId":"the account","duration":"3600s"}`))

	require.NoError(t, err)
	assert.False(t, result.Refused)
	assert.JSONEq(t, `{"accountId":"the account","offence":2}`, result.Text)
	assert.Equal(t, "Bearer cpat_ada", *authorization, "the gate admits the call as the app that holds the token")
}

func TestARefusalIsItsCodeAndItsSentence(t *testing.T) {
	caller, _ := serve(t)

	for _, arguments := range []string{`{}`, ``, `null`} {
		result, err := caller.Call(t.Context(), planetv1connect.AdminServiceBanPlayerProcedure, "cpat_ada", json.RawMessage(arguments))
		require.NoError(t, err, arguments)
		assert.Equal(t, tools.Result{Text: "invalid_argument: a scope or an account", Refused: true}, result, arguments)
	}

	result, err := caller.Call(t.Context(), planetv1connect.AdminServiceBanPlayerProcedure, "cpat_ada", json.RawMessage(`{"nope":1}`))
	require.NoError(t, err)
	assert.True(t, result.Refused, "protojson refuses a field the request does not have")
	assert.Contains(t, result.Text, "invalid_argument")

	result, err = caller.Call(t.Context(), planetv1connect.AdminServiceTopPlayersProcedure, "cpat_ada", nil)
	require.NoError(t, err)
	assert.Equal(t, "unimplemented: planet.v1.AdminService.TopPlayers is not implemented", result.Text)
}

func TestAListenerThatIsNotConnectIsAFailure(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(server.Close)

	_, err := connect_admin_caller.New(server.Client(), server.URL).Call(t.Context(), "/nope", "cpat_ada", nil)

	require.ErrorContains(t, err, "Not Found")
}

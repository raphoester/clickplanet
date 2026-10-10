package opsv1controller_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	opsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/ops/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/ops/v1/opsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/ops/internal/access"
	"github.com/raphoester/clickplanet.lol-backend/internal/ops/internal/opsv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/ops/internal/opsv1controller/query_handler"
)

type knownAssertions map[string]access.Caller

func (k knownAssertions) Caller(_ context.Context, assertion string) (access.Caller, error) {
	caller, known := k[assertion]
	if !known {
		return "", errors.New("nobody signed this assertion")
	}

	return caller, nil
}

type recordingQuery struct {
	callers *[]access.Caller
}

func (r recordingQuery) Rows(ctx context.Context, _ string, _ uint32) (*opsv1.QueryResponse, error) {
	*r.callers = append(*r.callers, access.CallerOf(ctx))
	return &opsv1.QueryResponse{Columns: []string{"answer"}}, nil
}

func serve(t *testing.T) (opsv1connect.OpsServiceClient, *[]access.Caller) {
	t.Helper()

	callers := &[]access.Caller{}
	service := opsv1controller.OpsService{QueryHandler: query_handler.New(recordingQuery{callers: callers})}
	mux := http.NewServeMux()
	mux.Handle(opsv1connect.NewOpsServiceHandler(service, connect.WithInterceptors(
		opsv1controller.NewAccessInterceptor(knownAssertions{"signed-by-access": "claude-cloud.access"}),
	)))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return opsv1connect.NewOpsServiceClient(server.Client(), server.URL), callers
}

func TestACallerAccessLetInReachesTheQueryUnderItsName(t *testing.T) {
	client, callers := serve(t)
	req := connect.NewRequest(&opsv1.QueryRequest{Statement: "SELECT 1"})
	req.Header().Set("Cf-Access-Jwt-Assertion", "signed-by-access")

	res, err := client.Query(t.Context(), req)

	require.NoError(t, err)
	assert.Equal(t, []string{"answer"}, res.Msg.GetColumns())
	assert.Equal(t, []access.Caller{"claude-cloud.access"}, *callers)
}

func TestARequestWithNoAssertionIsUnauthenticatedAndReadsNothing(t *testing.T) {
	client, callers := serve(t)

	_, err := client.Query(t.Context(), connect.NewRequest(&opsv1.QueryRequest{Statement: "SELECT 1"}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.Empty(t, *callers)
}

func TestAnAssertionNobodySignedIsUnauthenticatedAndSaysNothingOfWhy(t *testing.T) {
	client, callers := serve(t)
	req := connect.NewRequest(&opsv1.QueryRequest{Statement: "SELECT 1"})
	req.Header().Set("Cf-Access-Jwt-Assertion", "forged")

	_, err := client.Query(t.Context(), req)

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.ErrorContains(t, err, opsv1controller.ErrNoAccess.Error())
	assert.NotContains(t, err.Error(), "nobody signed")
	assert.Empty(t, *callers)
}

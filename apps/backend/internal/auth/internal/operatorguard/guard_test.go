package operatorguard_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/operatorguard"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/operatorguard/operator_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

const procedure = "/test.v1.AdminService/Do"

type operators map[string]error

func (o operators) Operator(_ context.Context, cookieHeader string) (string, error) {
	if err, refused := o[cookieHeader]; refused {
		return "", err
	}
	return "account of " + cookieHeader, nil
}

func serve(t *testing.T, seen *string) string {
	t.Helper()

	refusals := operators{
		"":                operator_query.ErrNoSession,
		"cp_sid=player":   operator_query.ErrNotAnOperator,
		"cp_sid=database": errors.New("postgres is down"),
	}
	handler := connect.NewUnaryHandler(procedure,
		func(ctx context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
			*seen = cpctx.GetOperator(ctx)
			return connect.NewResponse(&emptypb.Empty{}), nil
		},
		connect.WithInterceptors(operatorguard.New(refusals)),
	)

	router := http.NewServeMux()
	router.Handle(procedure, handler)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return server.URL
}

func callWith(t *testing.T, url, cookie string) error {
	t.Helper()

	request := connect.NewRequest(&emptypb.Empty{})
	if cookie != "" {
		request.Header().Set("Cookie", cookie)
	}
	_, err := connect.NewClient[emptypb.Empty, emptypb.Empty](http.DefaultClient, url+procedure).CallUnary(t.Context(), request)
	return err //nolint:wrapcheck // the tests read the connect code.
}

func TestTheGuardTellsTheHandlerWhichOperatorCalls(t *testing.T) {
	var seen string
	url := serve(t, &seen)

	require.NoError(t, callWith(t, url, "cp_sid=operator"))

	assert.Equal(t, "account of cp_sid=operator", seen)
}

func TestTheGuardRefusesWithTheCodeOfTheReason(t *testing.T) {
	var seen string
	url := serve(t, &seen)

	for cookie, code := range map[string]connect.Code{
		"":                connect.CodeUnauthenticated,
		"cp_sid=player":   connect.CodePermissionDenied,
		"cp_sid=database": connect.CodeUnknown,
	} {
		assert.Equal(t, code, connect.CodeOf(callWith(t, url, cookie)), cookie)
	}

	assert.Empty(t, seen, "no refused call reaches the handler")
}

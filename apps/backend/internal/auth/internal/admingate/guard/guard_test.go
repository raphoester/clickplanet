package guard_test

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

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admingate/admission"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admingate/guard"
)

const procedure = "/test.v1.AdminService/Do"

type admitter map[string]error

func (a admitter) Execute(_ context.Context, visit admission.Visit) (admission.Operator, error) {
	if err, refused := a[visit.Origin+" "+visit.Cookie]; refused {
		return "", err
	}
	return admission.Operator("the account of " + visit.Cookie), nil
}

func serve(t *testing.T, seen *admission.Operator) string {
	t.Helper()

	refusals := admitter{
		"page ":             admission.ErrNoSession,
		"game cp_sid=op":    admission.ErrForeignOrigin,
		"page cp_sid=bob":   admission.ErrNotAnOperator,
		"page cp_sid=crash": errors.New("auth is down"),
	}
	handler := connect.NewUnaryHandler(procedure,
		func(ctx context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
			*seen = guard.OperatorOf(ctx)
			return connect.NewResponse(&emptypb.Empty{}), nil
		},
		connect.WithInterceptors(guard.New(refusals)),
	)

	router := http.NewServeMux()
	router.Handle(procedure, handler)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return server.URL
}

func callFrom(t *testing.T, url, origin, cookie string) error {
	t.Helper()

	request := connect.NewRequest(&emptypb.Empty{})
	request.Header().Set("Origin", origin)
	request.Header().Set("Cookie", cookie)
	_, err := connect.NewClient[emptypb.Empty, emptypb.Empty](http.DefaultClient, url+procedure).CallUnary(t.Context(), request)
	return err //nolint:wrapcheck // the tests read the connect code.
}

func TestTheGuardTellsTheHandlerWhichOperatorCalls(t *testing.T) {
	var seen admission.Operator
	url := serve(t, &seen)

	require.NoError(t, callFrom(t, url, "page", "cp_sid=op"))

	assert.Equal(t, admission.Operator("the account of cp_sid=op"), seen)
}

func TestTheGuardRefusesWithTheCodeOfTheReason(t *testing.T) {
	var seen admission.Operator
	url := serve(t, &seen)

	for call, code := range map[[2]string]connect.Code{
		{"page", ""}:             connect.CodeUnauthenticated,
		{"game", "cp_sid=op"}:    connect.CodePermissionDenied,
		{"page", "cp_sid=bob"}:   connect.CodePermissionDenied,
		{"page", "cp_sid=crash"}: connect.CodeUnknown,
	} {
		assert.Equal(t, code, connect.CodeOf(callFrom(t, url, call[0], call[1])), call)
	}

	assert.Empty(t, seen, "no refused call reaches the handler")
}

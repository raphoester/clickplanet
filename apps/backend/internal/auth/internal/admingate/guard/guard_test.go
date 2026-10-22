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

type admitter struct {
	refusals map[string]error
	visits   []admission.Visit
}

func (a *admitter) Execute(_ context.Context, visit admission.Visit) (admission.Caller, error) {
	a.visits = append(a.visits, visit)
	if err, refused := a.refusals[visit.Origin+" "+visit.Cookie+" "+visit.Token]; refused {
		return admission.Caller{}, err
	}
	if visit.Token != "" {
		return admission.ConnectionOf(admission.Operator("the account of "+visit.Token), "Claude"), nil
	}
	return admission.SessionOf(admission.Operator("the account of " + visit.Cookie)), nil
}

func serve(t *testing.T, seen *admission.Caller) (string, *admitter) {
	t.Helper()

	admit := &admitter{refusals: map[string]error{
		"page  ":             admission.ErrNoSession,
		"game cp_sid=op ":    admission.ErrForeignOrigin,
		"page cp_sid=bob ":   admission.ErrNotAnOperator,
		"page cp_sid=crash ": errors.New("auth is down"),
		"  cpat_unknown":     admission.ErrNoToken,
		"  cpat_approve":     admission.ErrSessionOnly,
	}}
	handle := func(ctx context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
		*seen = guard.CallerOf(ctx)
		return connect.NewResponse(&emptypb.Empty{}), nil
	}

	router := http.NewServeMux()
	router.Handle(procedure, connect.NewUnaryHandler(procedure, handle, connect.WithInterceptors(guard.New(admit))))
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return server.URL, admit
}

func call(t *testing.T, url, called string, header http.Header) error {
	t.Helper()

	request := connect.NewRequest(&emptypb.Empty{})
	for name, values := range header {
		request.Header()[name] = values
	}
	_, err := connect.NewClient[emptypb.Empty, emptypb.Empty](http.DefaultClient, url+called).CallUnary(t.Context(), request)
	return err //nolint:wrapcheck // the tests read the connect code.
}

func from(origin, cookie string) http.Header {
	return http.Header{"Origin": {origin}, "Cookie": {cookie}}
}

func bearer(authorization string) http.Header {
	return http.Header{"Authorization": {authorization}}
}

func TestTheGuardTellsTheHandlerWhichOperatorCalls(t *testing.T) {
	var seen admission.Caller
	url, _ := serve(t, &seen)

	require.NoError(t, call(t, url, procedure, from("page", "cp_sid=op")))

	assert.Equal(t, admission.SessionOf("the account of cp_sid=op"), seen)
}

func TestTheGuardTellsTheHandlerWhichAppCalls(t *testing.T) {
	var seen admission.Caller
	url, _ := serve(t, &seen)

	require.NoError(t, call(t, url, procedure, bearer("Bearer cpat_ada")))

	assert.Equal(t, admission.ConnectionOf("the account of cpat_ada", "Claude"), seen)
}

func TestTheTokenIsReadFromABearerAuthorizationAlone(t *testing.T) {
	var seen admission.Caller
	url, admit := serve(t, &seen)

	for authorization, token := range map[string]string{
		"Bearer cpat_ada":   "cpat_ada",
		"bearer cpat_ada":   "cpat_ada",
		"Bearer  cpat_ada ": "cpat_ada",
		"Basic cpat_ada":    "",
		"cpat_ada":          "",
	} {
		admit.visits = nil
		_ = call(t, url, procedure, bearer(authorization))
		require.Len(t, admit.visits, 1, authorization)
		assert.Equal(t, token, admit.visits[0].Token, authorization)
	}
}

func TestTheGuardRefusesWithTheCodeOfTheReason(t *testing.T) {
	var seen admission.Caller
	url, _ := serve(t, &seen)

	for name, c := range map[string]struct {
		header http.Header
		code   connect.Code
	}{
		"no session":      {from("page", ""), connect.CodeUnauthenticated},
		"another page":    {from("game", "cp_sid=op"), connect.CodePermissionDenied},
		"a player":        {from("page", "cp_sid=bob"), connect.CodePermissionDenied},
		"auth is down":    {from("page", "cp_sid=crash"), connect.CodeUnknown},
		"no such token":   {bearer("Bearer cpat_unknown"), connect.CodeUnauthenticated},
		"the page's call": {bearer("Bearer cpat_approve"), connect.CodePermissionDenied},
	} {
		assert.Equal(t, c.code, connect.CodeOf(call(t, url, procedure, c.header)), name)
	}

	assert.Empty(t, seen, "no refused call reaches the handler")
}

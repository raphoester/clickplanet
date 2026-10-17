package log_calls_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/raphoester/clickplanet.lol-backend/internal/admingate/internal/admission"
	"github.com/raphoester/clickplanet.lol-backend/internal/admingate/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/admingate/internal/guard/log_calls"
)

const procedure = "/test.v1.AdminService/Do"

type admitAll struct{}

func (admitAll) Execute(context.Context, admission.Visit) (admission.Operator, error) {
	return "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10", nil
}

func TestEveryCallIsLoggedWithItsOperatorWhatItAskedAndItsOutcome(t *testing.T) {
	var logs bytes.Buffer
	handler := connect.NewUnaryHandler(procedure,
		func(_ context.Context, req *connect.Request[wrapperspb.StringValue]) (*connect.Response[wrapperspb.StringValue], error) {
			if req.Msg.GetValue() == "missing" {
				return nil, connect.NewError(connect.CodeNotFound, errors.New("no such ban"))
			}
			return connect.NewResponse(&wrapperspb.StringValue{}), nil
		},
		connect.WithInterceptors(guard.New(admitAll{}), log_calls.New(slog.New(slog.NewTextHandler(&logs, nil)))),
	)
	router := http.NewServeMux()
	router.Handle(procedure, handler)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	client := connect.NewClient[wrapperspb.StringValue, wrapperspb.StringValue](http.DefaultClient, server.URL+procedure)

	_, err := client.CallUnary(t.Context(), connect.NewRequest(wrapperspb.String("1.2.3.4")))
	require.NoError(t, err)
	_, err = client.CallUnary(t.Context(), connect.NewRequest(wrapperspb.String("missing")))
	require.Error(t, err)

	assert.Contains(t, logs.String(),
		`level=INFO msg="admin call" operator=0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10 procedure=/test.v1.AdminService/Do request="\"1.2.3.4\"" code=ok`)
	assert.Contains(t, logs.String(), `request="\"missing\"" code=not_found`)
}

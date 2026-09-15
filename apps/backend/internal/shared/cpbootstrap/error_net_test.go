package cpbootstrap_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"
)

const failingProcedure = "/test.v1.TestService/Do"

// The module below asks for no interceptors at all. That is the point: the net
// is the server's, so a module cannot leave it out by forgetting it, and a
// module added later gets it without knowing it exists.
func TestEveryMountedServiceGetsTheErrorNet(t *testing.T) {
	var handler http.Handler

	err := run(t, []cpbootstrap.Module{
		newModule("leaky", func(props cpbootstrap.Props) error {
			return props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
				handler = connect.NewUnaryHandler(failingProcedure,
					func(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
						return nil, errors.New("disk on fire")
					},
					options...,
				)

				return failingProcedure, handler
			})
		}),
	})
	require.NoError(t, err)
	require.NotNil(t, handler, "the module never mounted")

	mux := http.NewServeMux()
	mux.Handle(failingProcedure, handler)

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := connect.NewClient[emptypb.Empty, emptypb.Empty](
		server.Client(), server.URL+failingProcedure)

	_, callErr := client.CallUnary(t.Context(), connect.NewRequest(&emptypb.Empty{}))

	require.Equal(t, connect.CodeInternal, connect.CodeOf(callErr))
	require.NotContains(t, callErr.Error(), "disk on fire",
		"the cause is the server's business and must not reach the caller")
}

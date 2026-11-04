package cpbootstrap_test

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cphttpserver"
)

func TestEachRunOfTheProcessAnswersUnderABootOfItsOwn(t *testing.T) {
	first := startServer(t, answeringModule())
	firstBoot := bootOf(t, first)
	assert.Equal(t, firstBoot, bootOf(t, first), "one run answered under two boots")
	_, err := first.shutdown()
	require.NoError(t, err)

	second := startServer(t, answeringModule())
	secondBoot := bootOf(t, second)

	assert.NotEmpty(t, firstBoot)
	assert.NotEqual(t, firstBoot, secondBoot, "a restart kept the boot")
}

func answeringModule() cpbootstrap.Module {
	return newModule("planet", func(props cpbootstrap.Props) error {
		return props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
			return unaryProcedure, connect.NewUnaryHandler(unaryProcedure,
				func(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
					return connect.NewResponse(&emptypb.Empty{}), nil
				},
				options...,
			)
		})
	})
}

func bootOf(t *testing.T, server runningServer) string {
	t.Helper()

	client := connect.NewClient[emptypb.Empty, emptypb.Empty](http.DefaultClient, "http://"+server.address+unaryProcedure)
	response, err := client.CallUnary(t.Context(), connect.NewRequest(&emptypb.Empty{}))
	require.NoError(t, err)
	return response.Header().Get(cphttpserver.BootHeader)
}

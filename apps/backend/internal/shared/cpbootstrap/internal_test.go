package cpbootstrap_test

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

const (
	internalProcedure = "/callee.v1.InternalService/Answer"
	publicProcedure   = "/caller.v1.PublicService/Ask"
)

func calleeModule() cpbootstrap.Module {
	return newModule("callee", func(props cpbootstrap.Props) error {
		return props.InternalRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
			return internalProcedure, connect.NewUnaryHandler(internalProcedure,
				func(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[wrapperspb.StringValue], error) {
					return connect.NewResponse(wrapperspb.String("from the callee")), nil
				},
				options...,
			)
		})
	})
}

func callerModule() cpbootstrap.Module {
	return newModule("caller", func(props cpbootstrap.Props) error {
		httpClient, baseURL, err := props.Internal.Dial()
		if err != nil {
			return err //nolint:wrapcheck // the test reads the message.
		}
		callee := connect.NewClient[emptypb.Empty, wrapperspb.StringValue](httpClient, baseURL+internalProcedure)

		return props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
			return publicProcedure, connect.NewUnaryHandler(publicProcedure,
				func(ctx context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[wrapperspb.StringValue], error) {
					answer, err := callee.CallUnary(ctx, connect.NewRequest(&emptypb.Empty{}))
					if err != nil {
						return nil, err //nolint:wrapcheck // passed on as is.
					}
					return connect.NewResponse(answer.Msg), nil
				},
				options...,
			)
		})
	})
}

func ask(ctx context.Context, address, procedure string) (string, error) {
	client := connect.NewClient[emptypb.Empty, wrapperspb.StringValue](http.DefaultClient, "http://"+address+procedure)
	res, err := client.CallUnary(ctx, connect.NewRequest(&emptypb.Empty{}))
	if err != nil {
		return "", err //nolint:wrapcheck // the test reads the connect code.
	}
	return res.Msg.GetValue(), nil
}

func TestAModuleCallsAnotherOverTheInternalListener(t *testing.T) {
	server := cpbootstrap.ServerConfig{BindAddress: freeAddress(t), InternalBindAddress: freeAddress(t)}

	serveUntil(t, server, func() {
		answer, err := ask(t.Context(), server.BindAddress, publicProcedure)
		require.NoError(t, err)
		assert.Equal(t, "from the callee", answer)
	}, calleeModule(), callerModule())
}

func TestAnInternalServiceIsNotOnThePublicRouter(t *testing.T) {
	server := cpbootstrap.ServerConfig{BindAddress: freeAddress(t), InternalBindAddress: freeAddress(t)}

	serveUntil(t, server, func() {
		_, err := ask(t.Context(), server.InternalBindAddress, internalProcedure)
		require.NoError(t, err)

		_, err = ask(t.Context(), server.BindAddress, internalProcedure)
		assert.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
	}, calleeModule())
}

func TestDialingWithNoInternalListenerRefusesTheBoot(t *testing.T) {
	err := cpbootstrap.Run(t.Context(), cpbootstrap.Options{
		Server:  cpbootstrap.ServerConfig{BindAddress: "127.0.0.1:0"},
		Logger:  slog.New(slog.DiscardHandler),
		Modules: []cpbootstrap.Module{callerModule()},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "internalBindAddress")
}

func TestOnlyALoopbackInternalAddressIsAccepted(t *testing.T) {
	for address, ok := range map[string]bool{
		"":               true,
		"127.0.0.1:8082": true,
		"[::1]:8082":     true,
		":8082":          false,
		"0.0.0.0:8082":   false,
	} {
		err := cpbootstrap.ServerConfig{BindAddress: "0.0.0.0:8080", AllowedOrigin: "https://clickplanet.lol", InternalBindAddress: address}.Validate()
		if ok {
			require.NoError(t, err, address)
		} else {
			require.Error(t, err, address)
		}
	}
}

func TestATakenInternalAddressRefusesTheBoot(t *testing.T) {
	taken := freeAddress(t)
	blocker, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", taken)
	require.NoError(t, err)
	t.Cleanup(func() { _ = blocker.Close() })

	err = cpbootstrap.Run(t.Context(), cpbootstrap.Options{
		Server:  cpbootstrap.ServerConfig{BindAddress: "127.0.0.1:0", InternalBindAddress: taken},
		Logger:  slog.New(slog.DiscardHandler),
		Modules: []cpbootstrap.Module{calleeModule()},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "internalBindAddress")
}

package cpbootstrap_test

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

const adminProcedure = "/test.v1.AdminService/Do"

func freeAddress(t *testing.T) string {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())

	return address
}

func adminModule(result error) cpbootstrap.Module {
	return newModule("planet", func(props cpbootstrap.Props) error {
		return props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
			return adminProcedure, connect.NewUnaryHandler(adminProcedure,
				func(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
					if result != nil {
						return nil, result
					}
					return connect.NewResponse(&emptypb.Empty{}), nil
				},
				options...,
			)
		})
	})
}

func call(ctx context.Context, address string) error {
	client := connect.NewClient[emptypb.Empty, emptypb.Empty](http.DefaultClient, "http://"+address+adminProcedure)
	_, err := client.CallUnary(ctx, connect.NewRequest(&emptypb.Empty{}))
	return err //nolint:wrapcheck // the test reads the connect code.
}

func serveUntil(t *testing.T, server cpbootstrap.ServerConfig, probe func(), modules ...cpbootstrap.Module) {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- cpbootstrap.Run(ctx, cpbootstrap.Options{
			Server:  server,
			Logger:  slog.New(slog.DiscardHandler),
			Modules: modules,
		})
	}()

	require.Eventually(t, func() bool {
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", server.BindAddress)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, 5*time.Second, 10*time.Millisecond, "the server never came up")

	probe()

	cancel()
	require.NoError(t, <-done)
}

func TestAnAdminServiceIsServedOnTheAdminListenerAndNowhereElse(t *testing.T) {
	server := cpbootstrap.ServerConfig{BindAddress: freeAddress(t), AdminBindAddress: freeAddress(t)}

	serveUntil(t, server, func() {
		require.NoError(t, call(t.Context(), server.AdminBindAddress))
		assert.Equal(t, connect.CodeUnimplemented, connect.CodeOf(call(t.Context(), server.BindAddress)),
			"an admin service on the public router is one proxy edit away from the internet")
	}, adminModule(nil))
}

func TestAnAdminServiceGetsTheErrorNet(t *testing.T) {
	server := cpbootstrap.ServerConfig{BindAddress: freeAddress(t), AdminBindAddress: freeAddress(t)}

	serveUntil(t, server, func() {
		err := call(t.Context(), server.AdminBindAddress)
		assert.Equal(t, connect.CodeInternal, connect.CodeOf(err))
		assert.NotContains(t, err.Error(), "disk on fire")
	}, adminModule(errors.New("disk on fire")))
}

func TestNoAdminAddressServesNoAdminListener(t *testing.T) {
	admin := freeAddress(t)
	server := cpbootstrap.ServerConfig{BindAddress: freeAddress(t)}

	serveUntil(t, server, func() {
		assert.Equal(t, connect.CodeUnavailable, connect.CodeOf(call(t.Context(), admin)))
	}, adminModule(nil))
}

func TestOnlyALoopbackAdminAddressIsAccepted(t *testing.T) {
	for address, ok := range map[string]bool{
		"":               true,
		"127.0.0.1:8081": true,
		"localhost:8081": true,
		"[::1]:8081":     true,
		":8081":          false,
		"0.0.0.0:8081":   false,
		"10.0.0.5:8081":  false,
		"[::]:8081":      false,
		"127.0.0.1":      false,
	} {
		err := cpbootstrap.ServerConfig{BindAddress: "0.0.0.0:8080", AdminBindAddress: address}.Validate()
		if ok {
			require.NoError(t, err, address)
		} else {
			require.Error(t, err, address)
		}
	}
}

func TestATakenAdminAddressRefusesTheBoot(t *testing.T) {
	taken, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = taken.Close() })

	err = cpbootstrap.Run(t.Context(), cpbootstrap.Options{
		Server:  cpbootstrap.ServerConfig{BindAddress: "127.0.0.1:0", AdminBindAddress: taken.Addr().String()},
		Logger:  slog.New(slog.DiscardHandler),
		Modules: []cpbootstrap.Module{adminModule(nil)},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "adminBindAddress")
}

func TestRunRefusesANonLoopbackAdminAddressEvenUnvalidated(t *testing.T) {
	err := cpbootstrap.Run(t.Context(), cpbootstrap.Options{
		Server:  cpbootstrap.ServerConfig{BindAddress: "127.0.0.1:0", AdminBindAddress: "0.0.0.0:0"},
		Logger:  slog.New(slog.DiscardHandler),
		Modules: []cpbootstrap.Module{adminModule(nil)},
	})

	require.Error(t, err)
}

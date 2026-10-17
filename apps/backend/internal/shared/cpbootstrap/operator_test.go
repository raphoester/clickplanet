package cpbootstrap_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

const (
	operatorHeader = "X-Test-Operator"
	adminOrigin    = "https://admin.clickplanet.lol"
)

func headerGuard() connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
			operator := request.Header().Get(operatorHeader)
			if operator == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("who are you"))
			}

			return next(cpctx.AddOperatorToContext(ctx, operator), request)
		}
	})
}

func guardModule(interceptors ...connect.Interceptor) cpbootstrap.Module {
	return newModule("backoffice", func(props cpbootstrap.Props) error {
		return props.OperatorRPC.Guard(interceptors...)
	})
}

type seenOperators struct {
	mu   sync.Mutex
	seen []string
}

func (s *seenOperators) all() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.seen...)
}

func recordingAdminModule(seen *seenOperators) cpbootstrap.Module {
	return newModule("planet", func(props cpbootstrap.Props) error {
		return props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
			return adminProcedure, connect.NewUnaryHandler(adminProcedure,
				func(ctx context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
					seen.mu.Lock()
					seen.seen = append(seen.seen, cpctx.GetOperator(ctx))
					seen.mu.Unlock()
					return connect.NewResponse(&emptypb.Empty{}), nil
				},
				options...,
			)
		})
	})
}

func callAs(ctx context.Context, address, origin, operator string) error {
	client := connect.NewClient[emptypb.Empty, emptypb.Empty](http.DefaultClient, "http://"+address+adminProcedure)
	request := connect.NewRequest(&emptypb.Empty{})
	request.Header().Set("Origin", origin)
	request.Header().Set(operatorHeader, operator)
	_, err := client.CallUnary(ctx, request)
	return err //nolint:wrapcheck // the test reads the connect code.
}

func operatorServer(public, operator net.Listener) cpbootstrap.ServerConfig {
	return cpbootstrap.ServerConfig{
		BindAddress:         public.Addr().String(),
		OperatorBindAddress: operator.Addr().String(),
		OperatorOrigin:      adminOrigin,
	}
}

func TestTheOperatorListenerServesTheAdminServicesBehindTheGuard(t *testing.T) {
	public, admin, operator := listen(t), listen(t), listen(t)
	server := operatorServer(public, operator)
	server.AdminBindAddress = admin.Addr().String()
	seen := &seenOperators{}

	serveUntil(t, server, []net.Listener{public, admin, operator}, func() {
		assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(callAs(t.Context(), server.OperatorBindAddress, adminOrigin, "")))
		require.NoError(t, callAs(t.Context(), server.OperatorBindAddress, adminOrigin, "the operator"))
		require.NoError(t, call(t.Context(), server.AdminBindAddress), "the loopback listener stays as it was")
		assert.Equal(t, connect.CodeUnimplemented, connect.CodeOf(call(t.Context(), server.BindAddress)))
	}, recordingAdminModule(seen), guardModule(headerGuard()))

	assert.Equal(t, []string{"the operator", ""}, seen.all(), "the handler is told who called, and nobody on the loopback")
}

func TestTheOperatorListenerAnswersTheAdminPageOnly(t *testing.T) {
	public, operator := listen(t), listen(t)
	server := operatorServer(public, operator)
	seen := &seenOperators{}

	serveUntil(t, server, []net.Listener{public, operator}, func() {
		for _, origin := range []string{"", "https://clickplanet.lol", "https://admin.clickplanet.lol.evil.example", "http://admin.clickplanet.lol"} {
			err := callAs(t.Context(), server.OperatorBindAddress, origin, "the operator")
			assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), origin)
		}
	}, recordingAdminModule(seen), guardModule(headerGuard()))

	assert.Empty(t, seen.all(), "the game page holds the cookie too, so its origin is refused before the guard")
}

func TestTheOperatorListenerAnswersThePreflightOfTheAdminPage(t *testing.T) {
	public, operator := listen(t), listen(t)
	server := operatorServer(public, operator)

	serveUntil(t, server, []net.Listener{public, operator}, func() {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodOptions, "http://"+server.OperatorBindAddress+adminProcedure, nil)
		require.NoError(t, err)
		req.Header.Set("Origin", adminOrigin)
		req.Close = true
		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.NoError(t, res.Body.Close())

		assert.Equal(t, adminOrigin, res.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", res.Header.Get("Access-Control-Allow-Credentials"))
	}, adminModule(nil), guardModule(headerGuard()))
}

func TestEveryOperatorCallIsLoggedWithItsOperator(t *testing.T) {
	public, operator := listen(t), listen(t)
	server := operatorServer(public, operator)
	logs := &lockedBuffer{}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- cpbootstrap.RunOn(ctx, cpbootstrap.Options{
			Server:  server,
			Logger:  slog.New(slog.NewTextHandler(logs, nil)),
			Modules: []cpbootstrap.Module{adminModule(nil), guardModule(headerGuard())},
		}, public, operator)
	}()
	waitUntilServed(t, server.BindAddress)

	require.NoError(t, callAs(t.Context(), server.OperatorBindAddress, adminOrigin, "the-operator"))

	cancel()
	require.NoError(t, <-done)
	assert.Contains(t, logs.String(), `msg="operator call" operator=the-operator procedure=/test.v1.AdminService/Do code=ok`)
}

func TestTheOperatorListenerServesOnlyTheAdminServices(t *testing.T) {
	public, operator := listen(t), listen(t)
	server := operatorServer(public, operator)

	serveUntil(t, server, []net.Listener{public, operator}, func() {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+server.OperatorBindAddress+"/metrics", nil)
		require.NoError(t, err)
		req.Close = true
		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.NoError(t, res.Body.Close())

		assert.Equal(t, http.StatusNotFound, res.StatusCode)
	}, adminModule(nil), guardModule(headerGuard()))
}

func TestTheGuardIsInsideTheErrorNet(t *testing.T) {
	public, operator := listen(t), listen(t)
	server := operatorServer(public, operator)
	broken := connect.UnaryInterceptorFunc(func(connect.UnaryFunc) connect.UnaryFunc {
		return func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
			return nil, errors.New("access keys on fire")
		}
	})

	serveUntil(t, server, []net.Listener{public, operator}, func() {
		err := callAs(t.Context(), server.OperatorBindAddress, adminOrigin, "the operator")
		assert.Equal(t, connect.CodeInternal, connect.CodeOf(err))
		assert.NotContains(t, err.Error(), "on fire")
	}, adminModule(nil), guardModule(broken))
}

func TestAnOperatorAddressNobodyGuardsRefusesTheBoot(t *testing.T) {
	public, operator := listen(t), listen(t)

	err := cpbootstrap.RunOn(t.Context(), cpbootstrap.Options{
		Server:  operatorServer(public, operator),
		Logger:  slog.New(slog.DiscardHandler),
		Modules: []cpbootstrap.Module{adminModule(nil)},
	}, public, operator)

	require.ErrorContains(t, err, "no module guards it")
}

func TestOneModuleGuardsTheOperatorListener(t *testing.T) {
	err := cpbootstrap.Run(t.Context(), cpbootstrap.Options{
		Server:  cpbootstrap.ServerConfig{BindAddress: "127.0.0.1:0"},
		Logger:  slog.New(slog.DiscardHandler),
		Modules: []cpbootstrap.Module{guardModule(headerGuard()), guardModule(headerGuard())},
	})

	require.ErrorContains(t, err, "already guards")
}

func TestAGuardOfNothingRefusesTheBoot(t *testing.T) {
	err := cpbootstrap.Run(t.Context(), cpbootstrap.Options{
		Server:  cpbootstrap.ServerConfig{BindAddress: "127.0.0.1:0"},
		Logger:  slog.New(slog.DiscardHandler),
		Modules: []cpbootstrap.Module{guardModule()},
	})

	require.ErrorContains(t, err, "no interceptor")
}

func TestTheOperatorAddressIsAHostAndAPort(t *testing.T) {
	for address, ok := range map[string]bool{
		"":             true,
		"0.0.0.0:8083": true,
		":8083":        true,
		"backend":      false,
		"0.0.0.0:":     false,
	} {
		err := cpbootstrap.ServerConfig{
			BindAddress: "0.0.0.0:8080", AllowedOrigin: "https://clickplanet.lol",
			OperatorBindAddress: address, OperatorOrigin: adminOrigin,
		}.Validate()
		if ok {
			require.NoError(t, err, address)
		} else {
			require.ErrorContains(t, err, "operatorBindAddress", address)
		}
	}
}

func TestAnOperatorListenerNeedsTheOriginOfThePageThatCalls(t *testing.T) {
	for origin, ok := range map[string]bool{
		adminOrigin:                      true,
		"http://localhost:5174":          true,
		"":                               false,
		"*":                              false,
		"https://admin.clickplanet.lol/": false,
	} {
		err := cpbootstrap.ServerConfig{
			BindAddress: "0.0.0.0:8080", AllowedOrigin: "https://clickplanet.lol",
			OperatorBindAddress: "0.0.0.0:8083", OperatorOrigin: origin,
		}.Validate()
		if ok {
			require.NoError(t, err, origin)
		} else {
			require.ErrorContains(t, err, "operatorOrigin", origin)
		}
	}

	require.NoError(t, cpbootstrap.ServerConfig{BindAddress: "0.0.0.0:8080", AllowedOrigin: "https://clickplanet.lol"}.Validate(),
		"no operator listener needs no origin")
}

func TestRunRefusesAnOperatorListenerWithNoOriginEvenUnvalidated(t *testing.T) {
	public, operator := listen(t), listen(t)
	server := operatorServer(public, operator)
	server.OperatorOrigin = ""

	err := cpbootstrap.RunOn(t.Context(), cpbootstrap.Options{
		Server:  server,
		Logger:  slog.New(slog.DiscardHandler),
		Modules: []cpbootstrap.Module{adminModule(nil), guardModule(headerGuard())},
	}, public, operator)

	require.ErrorContains(t, err, "operatorOrigin")
}

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p) //nolint:wrapcheck // a buffer's write never fails.
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

package cpbootstrap_test

import (
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
)

const (
	passHeader   = "X-Test-Pass"
	gateHeader   = "X-Test-Gate"
	callerHeader = "X-Test-Caller"
)

type callerKey struct{}

func headerGate() cpbootstrap.Gate {
	return cpbootstrap.Gate{
		Middlewares: []func(http.Handler) http.Handler{func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(gateHeader, "yes")
				next.ServeHTTP(w, r)
			})
		}},
		Interceptors: []connect.Interceptor{connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
			return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
				pass := request.Header().Get(passHeader)
				if pass == "" {
					return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("no pass"))
				}
				return next(context.WithValue(ctx, callerKey{}, pass), request)
			}
		})},
	}
}

type seenCallers struct {
	mu   sync.Mutex
	seen []string
}

func (s *seenCallers) all() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.seen...)
}

func recordingAdminModule(seen *seenCallers) cpbootstrap.Module {
	return newModule("planet", func(props cpbootstrap.Props) error {
		return props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
			return adminProcedure, connect.NewUnaryHandler(adminProcedure,
				func(ctx context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
					caller, _ := ctx.Value(callerKey{}).(string)
					seen.mu.Lock()
					seen.seen = append(seen.seen, caller)
					seen.mu.Unlock()
					return connect.NewResponse(&emptypb.Empty{}), nil
				},
				options...,
			)
		})
	})
}

func callWithPass(ctx context.Context, address, pass string) (http.Header, error) {
	client := connect.NewClient[emptypb.Empty, emptypb.Empty](http.DefaultClient, "http://"+address+adminProcedure)
	request := connect.NewRequest(&emptypb.Empty{})
	request.Header().Set(passHeader, pass)
	response, err := client.CallUnary(ctx, request)
	if err != nil {
		return nil, err //nolint:wrapcheck // the test reads the connect code.
	}
	return response.Header(), nil
}

func serveBehind(t *testing.T, server cpbootstrap.ServerConfig, gate cpbootstrap.Gate, listeners []net.Listener, probe func(), modules ...cpbootstrap.Module) {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- cpbootstrap.RunOn(ctx, cpbootstrap.Options{
			Server:    server,
			Logger:    slog.New(slog.DiscardHandler),
			Modules:   modules,
			AdminGate: gate,
		}, listeners...)
	}()

	waitUntilServed(t, server.BindAddress)

	probe()

	cancel()
	require.NoError(t, <-done)
}

func TestTheAdminListenerServesTheAdminServicesBehindTheGate(t *testing.T) {
	public, console, admin := listen(t), listen(t), listen(t)
	server := cpbootstrap.ServerConfig{
		BindAddress:        public.Addr().String(),
		ConsoleBindAddress: console.Addr().String(),
		AdminBindAddress:   admin.Addr().String(),
	}
	seen := &seenCallers{}

	serveBehind(t, server, headerGate(), []net.Listener{public, console, admin}, func() {
		_, err := callWithPass(t.Context(), server.AdminBindAddress, "")
		assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

		header, err := callWithPass(t.Context(), server.AdminBindAddress, "the caller")
		require.NoError(t, err)
		assert.Equal(t, "yes", header.Get(gateHeader), "the gate's middlewares wrap the router")

		require.NoError(t, call(t.Context(), server.ConsoleBindAddress), "the console listener stays as it was")
		assert.Equal(t, connect.CodeUnimplemented, connect.CodeOf(call(t.Context(), server.BindAddress)))
	}, recordingAdminModule(seen))

	assert.Equal(t, []string{"the caller", ""}, seen.all(), "the gate's context reaches the handler, and nothing on the loopback")
}

func TestTheAdminListenerServesOnlyTheAdminServices(t *testing.T) {
	public, admin := listen(t), listen(t)
	server := cpbootstrap.ServerConfig{BindAddress: public.Addr().String(), AdminBindAddress: admin.Addr().String()}

	serveBehind(t, server, headerGate(), []net.Listener{public, admin}, func() {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+server.AdminBindAddress+"/metrics", nil)
		require.NoError(t, err)
		req.Close = true
		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.NoError(t, res.Body.Close())

		assert.Equal(t, http.StatusNotFound, res.StatusCode)
	}, adminModule(nil))
}

func TestTheGateIsInsideTheErrorNet(t *testing.T) {
	public, admin := listen(t), listen(t)
	server := cpbootstrap.ServerConfig{BindAddress: public.Addr().String(), AdminBindAddress: admin.Addr().String()}
	broken := cpbootstrap.Gate{Interceptors: []connect.Interceptor{connect.UnaryInterceptorFunc(func(connect.UnaryFunc) connect.UnaryFunc {
		return func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
			return nil, errors.New("the gate is on fire")
		}
	})}}

	serveBehind(t, server, broken, []net.Listener{public, admin}, func() {
		_, err := callWithPass(t.Context(), server.AdminBindAddress, "the caller")
		assert.Equal(t, connect.CodeInternal, connect.CodeOf(err))
		assert.NotContains(t, err.Error(), "on fire")
	}, adminModule(nil))
}

func TestAnAdminAddressWithNoGateRefusesTheBoot(t *testing.T) {
	public, admin := listen(t), listen(t)

	err := cpbootstrap.RunOn(t.Context(), cpbootstrap.Options{
		Server:  cpbootstrap.ServerConfig{BindAddress: public.Addr().String(), AdminBindAddress: admin.Addr().String()},
		Logger:  slog.New(slog.DiscardHandler),
		Modules: []cpbootstrap.Module{adminModule(nil)},
	}, public, admin)

	require.ErrorContains(t, err, "nothing gates it")
}

func TestAGateHoldingNothingUsableRefusesTheBoot(t *testing.T) {
	public, admin := listen(t), listen(t)
	gate := headerGate()
	gate.Interceptors = append(gate.Interceptors, nil)

	err := cpbootstrap.RunOn(t.Context(), cpbootstrap.Options{
		Server:    cpbootstrap.ServerConfig{BindAddress: public.Addr().String(), AdminBindAddress: admin.Addr().String()},
		Logger:    slog.New(slog.DiscardHandler),
		Modules:   []cpbootstrap.Module{adminModule(nil)},
		AdminGate: gate,
	}, public, admin)

	require.ErrorContains(t, err, "nil interceptor")
}

func TestTheAdminAddressIsAHostAndAPort(t *testing.T) {
	for address, ok := range map[string]bool{
		"":             true,
		"0.0.0.0:8083": true,
		":8083":        true,
		"backend":      false,
		"0.0.0.0:":     false,
	} {
		err := cpbootstrap.ServerConfig{
			BindAddress: "0.0.0.0:8080", AllowedOrigin: "https://clickplanet.lol", AdminBindAddress: address,
		}.Validate()
		if ok {
			require.NoError(t, err, address)
		} else {
			require.ErrorContains(t, err, "adminBindAddress", address)
		}
	}
}

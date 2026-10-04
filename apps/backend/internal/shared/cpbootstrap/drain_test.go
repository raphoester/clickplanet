package cpbootstrap_test

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

const (
	streamProcedure = "/test.v1.LiveService/Listen"
	unaryProcedure  = "/test.v1.LiveService/Do"

	shutdownTimeout = 5 * time.Second
)

func TestAnOpenStreamEndsCleanlyAndDoesNotHoldTheShutdown(t *testing.T) {
	var streamReturned, closedAfterStream atomic.Bool

	module := newModule("planet", func(props cpbootstrap.Props) error {
		props.Closers.Add("snapshot", func() error {
			closedAfterStream.Store(streamReturned.Load())
			return nil
		})

		return props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
			return streamProcedure, connect.NewServerStreamHandler(streamProcedure,
				func(ctx context.Context, _ *connect.Request[emptypb.Empty], stream *connect.ServerStream[emptypb.Empty]) error {
					defer streamReturned.Store(true)

					if err := stream.Send(&emptypb.Empty{}); err != nil {
						return err //nolint:wrapcheck // the test reads what the client sees.
					}
					<-ctx.Done()
					return nil
				},
				options...,
			)
		})
	})

	server := startServer(t, module)

	client := connect.NewClient[emptypb.Empty, emptypb.Empty](http.DefaultClient, "http://"+server.address+streamProcedure)
	stream, err := client.CallServerStream(t.Context(), connect.NewRequest(&emptypb.Empty{}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = stream.Close() })
	require.True(t, stream.Receive(), "the stream never opened: %v", stream.Err())

	took, err := server.shutdown()

	require.NoError(t, err)
	assert.Less(t, took, time.Second, "the shutdown waited on the open stream")
	assert.Empty(t, server.errors(), "the shutdown logged an error")

	assert.False(t, stream.Receive(), "the stream is still open")
	require.NoError(t, stream.Err(), "the stream was cut rather than ended")

	assert.True(t, closedAfterStream.Load(), "a closer ran before the stream had ended")
}

func TestAUnaryCallInFlightFinishesDuringTheShutdown(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var cancelled atomic.Bool

	module := newModule("planet", func(props cpbootstrap.Props) error {
		return props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
			return unaryProcedure, connect.NewUnaryHandler(unaryProcedure,
				func(ctx context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
					close(entered)
					<-release
					cancelled.Store(ctx.Err() != nil)
					return connect.NewResponse(&emptypb.Empty{}), nil
				},
				options...,
			)
		})
	})

	server := startServer(t, module)

	answered := make(chan error, 1)
	go func() {
		client := connect.NewClient[emptypb.Empty, emptypb.Empty](http.DefaultClient, "http://"+server.address+unaryProcedure)
		_, err := client.CallUnary(t.Context(), connect.NewRequest(&emptypb.Empty{}))
		answered <- err
	}()
	<-entered

	stopped := make(chan error, 1)
	go func() {
		_, err := server.shutdown()
		stopped <- err
	}()

	time.Sleep(100 * time.Millisecond)
	close(release)

	require.NoError(t, <-answered)
	require.NoError(t, <-stopped)
	assert.False(t, cancelled.Load(), "the shutdown cancelled a unary call")
	assert.Empty(t, server.errors(), "the shutdown logged an error")
}

type runningServer struct {
	address  string
	cancel   context.CancelFunc
	done     chan error
	recorded *errorRecorder
}

func startServer(t *testing.T, module cpbootstrap.Module) runningServer {
	t.Helper()

	public := listen(t)
	address := public.Addr().String()
	recorded := &errorRecorder{}

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	done := make(chan error, 1)
	go func() {
		done <- cpbootstrap.RunOn(ctx, cpbootstrap.Options{
			Server:          cpbootstrap.ServerConfig{BindAddress: address},
			ShutdownTimeout: shutdownTimeout,
			Logger:          slog.New(recorded),
			Modules:         []cpbootstrap.Module{module},
		}, public)
	}()

	waitUntilServed(t, address)

	return runningServer{address: address, cancel: cancel, done: done, recorded: recorded}
}

func (s runningServer) shutdown() (time.Duration, error) {
	start := time.Now()
	s.cancel()
	err := <-s.done
	return time.Since(start), err
}

func (s runningServer) errors() []string {
	return s.recorded.messages()
}

type errorRecorder struct {
	mu     sync.Mutex
	logged []string
}

func (r *errorRecorder) Enabled(context.Context, slog.Level) bool { return true }

func (r *errorRecorder) Handle(_ context.Context, record slog.Record) error {
	if record.Level < slog.LevelError {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.logged = append(r.logged, record.Message)
	return nil
}

func (r *errorRecorder) WithAttrs([]slog.Attr) slog.Handler { return r }

func (r *errorRecorder) WithGroup(string) slog.Handler { return r }

func (r *errorRecorder) messages() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.logged...)
}

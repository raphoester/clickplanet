package cpbootstrap_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

func TestAModuleThatFailsToBuildNamesItselfInTheError(t *testing.T) {
	err := run(t, []cpbootstrap.Module{
		newModule("planet", func(cpbootstrap.Props) error { return nil }),
		newModule("chat", func(cpbootstrap.Props) error { return errors.New("no log path") }),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "chat")
	assert.Contains(t, err.Error(), "no log path")
}

func TestABootSlowerThanTheStartupTimeoutIsRefused(t *testing.T) {
	err := runOn(t, cpbootstrap.ServerConfig{BindAddress: "127.0.0.1:0", StartupTimeout: time.Millisecond},
		[]cpbootstrap.Module{newContextModule("planet", func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		})})

	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Contains(t, err.Error(), "planet")
}

func TestABootIsGivenAMinuteByDefault(t *testing.T) {
	var deadline time.Time
	began := time.Now()

	require.NoError(t, run(t, []cpbootstrap.Module{newContextModule("planet", func(ctx context.Context) error {
		deadline, _ = ctx.Deadline()
		return nil
	})}))

	assert.WithinDuration(t, began.Add(time.Minute), deadline, time.Second)
}

func TestTwoModulesCannotClaimTheSameRoute(t *testing.T) {
	var mountErr error

	err := run(t, []cpbootstrap.Module{
		newModule("planet", func(props cpbootstrap.Props) error {
			return props.RPC.Mount(mountOn("/planet.v1.ClickService/"))
		}),
		newModule("impostor", func(props cpbootstrap.Props) error {
			mountErr = props.RPC.Mount(mountOn("/planet.v1.ClickService/"))
			return mountErr
		}),
	})

	require.Error(t, err)
	require.Error(t, mountErr)
	assert.Contains(t, mountErr.Error(), "impostor")
	assert.Contains(t, mountErr.Error(), "planet")
}

func TestCleanupsRunInReverseRegistrationOrder(t *testing.T) {
	var (
		mu     sync.Mutex
		closed []string
	)
	record := func(name string) func() error {
		return func() error {
			mu.Lock()
			defer mu.Unlock()
			closed = append(closed, name)
			return nil
		}
	}

	require.NoError(t, run(t, []cpbootstrap.Module{
		newModule("first", func(props cpbootstrap.Props) error {
			props.Closers.Add("first", record("first"))
			return nil
		}),
		newModule("second", func(props cpbootstrap.Props) error {
			props.Closers.Add("second", record("second"))
			return nil
		}),
	}))

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"second", "first"}, closed)
}

func TestEveryCleanupRunsBeforeTheRunnersAreWaitedOn(t *testing.T) {
	stop := make(chan struct{})
	stopped := make(chan struct{})

	require.NoError(t, run(t, []cpbootstrap.Module{
		newModule("scheduled-job", func(props cpbootstrap.Props) error {
			props.Runners.Add(runner{name: "scheduled-job", run: func(context.Context) {
				<-stop
				close(stopped)
			}})
			props.Closers.Add("scheduled-job", func() error {
				close(stop)
				return nil
			})
			return nil
		}),
	}))

	select {
	case <-stopped:
	default:
		t.Fatal("the runner was still going when Run returned")
	}
}

func TestARunnerIsCancelledOnShutdown(t *testing.T) {
	cancelled := make(chan struct{})

	require.NoError(t, run(t, []cpbootstrap.Module{
		newModule("planet", func(props cpbootstrap.Props) error {
			props.Runners.Add(runner{name: "tiles-storage", run: func(ctx context.Context) {
				<-ctx.Done()
				close(cancelled)
			}})
			return nil
		}),
	}))

	select {
	case <-cancelled:
	default:
		t.Fatal("the runner was never cancelled")
	}
}

func TestAProcessWithNoModuleIsRefused(t *testing.T) {
	require.Error(t, run(t, nil))
}

func TestADisabledModuleIsNeverBuilt(t *testing.T) {
	built := false

	require.NoError(t, run(t, []cpbootstrap.Module{
		newModule("planet", func(cpbootstrap.Props) error { return nil }),
		disabled(newModule("session", func(cpbootstrap.Props) error {
			built = true
			return nil
		})),
	}))

	assert.False(t, built, "the disabled module's DI sequence ran")
}

func TestAProcessWhereEveryModuleIsOffIsRefused(t *testing.T) {
	err := run(t, []cpbootstrap.Module{
		disabled(newModule("planet", func(cpbootstrap.Props) error { return nil })),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "serves nothing")
}

func TestATakenBindAddressRefusesTheBoot(t *testing.T) {
	taken := listen(t)

	err := cpbootstrap.Run(t.Context(), cpbootstrap.Options{
		Server:  cpbootstrap.ServerConfig{BindAddress: taken.Addr().String()},
		Logger:  slog.New(slog.DiscardHandler),
		Modules: []cpbootstrap.Module{newModule("planet", func(cpbootstrap.Props) error { return nil })},
	})

	require.ErrorContains(t, err, "httpServer.bindAddress")
}

func TestRunOnBindsNoAddressItHoldsNoListenerFor(t *testing.T) {
	public := listen(t)

	err := cpbootstrap.RunOn(t.Context(), cpbootstrap.Options{
		Server:  cpbootstrap.ServerConfig{BindAddress: public.Addr().String(), InternalBindAddress: "127.0.0.1:0"},
		Logger:  slog.New(slog.DiscardHandler),
		Modules: []cpbootstrap.Module{newModule("planet", func(cpbootstrap.Props) error { return nil })},
	}, public)

	require.ErrorContains(t, err, "no listener held")
}

func newModule(name string, build func(cpbootstrap.Props) error) cpbootstrap.Module {
	return cpbootstrap.Module{
		Name:       name,
		Enabled:    true,
		DiSequence: func(_ context.Context, props cpbootstrap.Props) error { return build(props) },
	}
}

func newContextModule(name string, build func(context.Context) error) cpbootstrap.Module {
	return cpbootstrap.Module{
		Name:       name,
		Enabled:    true,
		DiSequence: func(ctx context.Context, _ cpbootstrap.Props) error { return build(ctx) },
	}
}

func disabled(module cpbootstrap.Module) cpbootstrap.Module {
	module.Enabled = false
	return module
}

func run(t *testing.T, modules []cpbootstrap.Module) error {
	t.Helper()

	return runOn(t, cpbootstrap.ServerConfig{BindAddress: "127.0.0.1:0"}, modules)
}

func runOn(t *testing.T, server cpbootstrap.ServerConfig, modules []cpbootstrap.Module) error {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	defer cancel()

	return cpbootstrap.Run(ctx, cpbootstrap.Options{
		Server:  server,
		Logger:  slog.New(slog.DiscardHandler),
		Modules: modules,
	})
}

func mountOn(path string) cpbootstrap.ServiceBuilder {
	return func(...connect.HandlerOption) (string, http.Handler) {
		return path, http.NotFoundHandler()
	}
}

type runner struct {
	name string
	run  func(ctx context.Context)
}

func (r runner) Name() string { return r.name }

func (r runner) Run(ctx context.Context) { r.run(ctx) }

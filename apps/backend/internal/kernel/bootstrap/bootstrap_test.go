package bootstrap_test

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/bootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging"
)

func TestAModuleThatFailsToBuildNamesItselfInTheError(t *testing.T) {
	err := run(t, []bootstrap.Module{
		newModule("clicks", func(bootstrap.Props) error { return nil }),
		newModule("chat", func(bootstrap.Props) error { return errors.New("no log path") }),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "chat")
	assert.Contains(t, err.Error(), "no log path")
}

// The router panics on a second registration for the same pattern, so the
// second module has to be refused before it gets there.
func TestTwoModulesCannotClaimTheSameRoute(t *testing.T) {
	var mountErr error

	err := run(t, []bootstrap.Module{
		newModule("clicks", func(props bootstrap.Props) error {
			return props.RPC.Mount("/planet.v1.ClickService/", http.NotFoundHandler())
		}),
		newModule("impostor", func(props bootstrap.Props) error {
			mountErr = props.RPC.Mount("/planet.v1.ClickService/", http.NotFoundHandler())
			return mountErr
		}),
	})

	require.Error(t, err)
	require.Error(t, mountErr)
	assert.Contains(t, mountErr.Error(), "impostor")
	assert.Contains(t, mountErr.Error(), "clicks")
}

// Registration order is dependency order, so reversing it closes a module
// before the ones it was built on top of.
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

	require.NoError(t, run(t, []bootstrap.Module{
		newModule("first", func(props bootstrap.Props) error {
			props.Closers.Add("first", record("first"))
			return nil
		}),
		newModule("second", func(props bootstrap.Props) error {
			props.Closers.Add("second", record("second"))
			return nil
		}),
	}))

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"second", "first"}, closed)
}

// A runner with no context of its own is stopped by a cleanup, so every cleanup
// has to have run before the runners are waited on.
func TestEveryCleanupRunsBeforeTheRunnersAreWaitedOn(t *testing.T) {
	stop := make(chan struct{})
	stopped := make(chan struct{})

	require.NoError(t, run(t, []bootstrap.Module{
		newModule("scheduled-job", func(props bootstrap.Props) error {
			props.Runners.Add("scheduled-job", func(context.Context) {
				<-stop
				close(stopped)
			})
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

	require.NoError(t, run(t, []bootstrap.Module{
		newModule("clicks", func(props bootstrap.Props) error {
			props.Runners.Add("tiles-storage", func(ctx context.Context) {
				<-ctx.Done()
				close(cancelled)
			})
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

func newModule(name string, build func(bootstrap.Props) error) bootstrap.Module {
	return bootstrap.Module{
		Name:       name,
		DiSequence: func(_ context.Context, props bootstrap.Props) error { return build(props) },
	}
}

// run boots on an ephemeral port and shuts down as soon as it is serving.
func run(t *testing.T, modules []bootstrap.Module) error {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	defer cancel()

	return bootstrap.Run(ctx, bootstrap.Options{
		BindAddress: "127.0.0.1:0",
		Logger:      logging.NewNopLogger(),
		Modules:     modules,
	})
}

package flags_test

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/postgres_flag_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/close_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/open_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpoutbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis/cpredistest"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func running(t *testing.T, runners ...cpbootstrap.Runner) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	var wg sync.WaitGroup
	for _, runner := range runners {
		wg.Go(func() { runner.Run(ctx) })
	}
	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})
}

func playable(t *testing.T, listener cpflags.Catalog, flag catalog.FlagID) func() bool {
	t.Helper()
	return func() bool {
		playable, err := listener.Playable(t.Context(), flag.String())
		require.NoError(t, err)
		return playable
	}
}

// What planet, the chat, scoring and player do: a catalog of their own, fed by what the flags module tells.
func TestAFlagOpenedIsPlayableInAnotherModuleWithNoRestartAndAClosedOneIsNot(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "flags", migrations.FS)
	redis := cpredistest.StartServer(t).Config()
	logger := slog.New(slog.DiscardHandler)

	teller := cpredis.New(redis, prometheus.NewRegistry())
	require.NoError(t, teller.ConnectCtx(t.Context()))
	t.Cleanup(func() { _ = teller.Close() })
	hearer := cpredis.New(redis, prometheus.NewRegistry())
	require.NoError(t, hearer.ConnectCtx(t.Context()))
	t.Cleanup(func() { _ = hearer.Close() })

	listener := cpflags.NewCatalog(cpflags.NewMemoryStore(), cpflags.Quiet{})
	opened, err := cpredis.Subscribe(t.Context(), hearer, "planet-flags-opened", cpflags.NewOpenedHandler(listener))
	require.NoError(t, err)
	closed, err := cpredis.Subscribe(t.Context(), hearer, "planet-flags-closed", cpflags.NewClosedHandler(listener))
	require.NoError(t, err)
	running(t, opened, closed, cpoutbox.NewRelay("flags-outbox", db, teller, logger))

	store := postgres_flag_store.New(db)
	clock := cptime.NewFixedClock(time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC))
	out, err := open_flag_usecase.New(&catalog.Sequence{}, clock, db, store, cpoutbox.New(db)).
		Execute(t.Context(), open_flag_usecase.In{Name: "Rennes"})
	require.NoError(t, err)

	require.Eventually(t, playable(t, listener, out.Flag), 10*time.Second, 20*time.Millisecond)

	clock.Advance(time.Hour)
	require.NoError(t, close_flag_usecase.New(clock, db, store, cpoutbox.New(db)).
		Execute(t.Context(), close_flag_usecase.In{Flag: out.Flag.String()}))

	require.Eventually(t, func() bool { return !playable(t, listener, out.Flag)() }, 10*time.Second, 20*time.Millisecond)
	known, err := listener.Known(t.Context(), out.Flag.String())
	require.NoError(t, err)
	require.True(t, known, "closed, never forgotten")
}

package flags_test

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/postgres_flag_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/close_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/open_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/rename_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed/usecases/tell_streams_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/subscribers/told_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
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

// heard keeps what another module hears of the flags, as planet's subscribers do.
type heard struct {
	mu    sync.Mutex
	facts []string
}

func (h *heard) Handle(_ context.Context, event proto.Message) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	switch fact := event.(type) {
	case *flagsv1.FlagOpened:
		h.facts = append(h.facts, "opened "+fact.GetFlag()+" "+fact.GetName())
	case *flagsv1.FlagClosed:
		h.facts = append(h.facts, "closed "+fact.GetFlag())
	}
	return nil
}

func (h *heard) all() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.facts)
}

type hearing[T proto.Message] struct{ heard *heard }

func (h hearing[T]) Handle(ctx context.Context, event T) error { return h.heard.Handle(ctx, event) }

// A flag opened or closed through the module is told, with no restart, to the modules that hear it and to the pages
// whose stream is open.
func TestAFlagOpenedOrClosedIsToldToTheOtherModulesAndToTheOpenPages(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "flags", migrations.FS)
	redis := cpredistest.StartServer(t).Config()
	logger := slog.New(slog.DiscardHandler)

	teller := cpredis.New(redis, prometheus.NewRegistry())
	require.NoError(t, teller.ConnectCtx(t.Context()))
	t.Cleanup(func() { _ = teller.Close() })
	hearer := cpredis.New(redis, prometheus.NewRegistry())
	require.NoError(t, hearer.ConnectCtx(t.Context()))
	t.Cleanup(func() { _ = hearer.Close() })

	planet := &heard{}
	opened, err := cpredis.Subscribe[*flagsv1.FlagOpened](t.Context(), hearer, "planet-flags-opened",
		hearing[*flagsv1.FlagOpened]{heard: planet})
	require.NoError(t, err)
	closed, err := cpredis.Subscribe[*flagsv1.FlagClosed](t.Context(), hearer, "planet-flags-closed",
		hearing[*flagsv1.FlagClosed]{heard: planet})
	require.NoError(t, err)

	store := postgres_flag_store.New(db)
	clock := cptime.NewFixedClock(time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC))
	board := feed.NewBoard(8)
	page, leave := board.Attend()
	t.Cleanup(leave)
	tell := tell_streams_usecase.New(feed.NewsSince(clock.Now()), store, board)
	streamsOpened, err := cpredis.Subscribe(t.Context(), teller, "flags-streams-opened", told_subscriber.NewOpened(tell))
	require.NoError(t, err)
	streamsClosed, err := cpredis.Subscribe(t.Context(), teller, "flags-streams-closed", told_subscriber.NewClosed(tell))
	require.NoError(t, err)
	streamsRenamed, err := cpredis.Subscribe(t.Context(), teller, "flags-streams-renamed", told_subscriber.NewRenamed(tell))
	require.NoError(t, err)
	running(t, opened, closed, streamsOpened, streamsClosed, streamsRenamed,
		cpoutbox.NewRelay("flags-outbox", db, teller, logger))

	bucket, upload := pictures.NewMemoryBucket(), pictures.UploadID{1}
	bucket.Upload(upload, pictures.TestPNG(300, 200))
	out, err := open_flag_usecase.New(catalog.NewOpening(&catalog.Sequence{}, clock),
		pictures.NewGallery(bucket, pictures.Rules{MaxBytes: 5 << 20}), db, store, cpoutbox.New(db)).
		Execute(t.Context(), open_flag_usecase.In{Name: "Rennes", Upload: upload.String(), EndsAt: clock.Now().Add(time.Hour)})
	require.NoError(t, err)
	require.Eventually(t, func() bool { return len(planet.all()) == 1 }, 10*time.Second, 20*time.Millisecond)
	assert.Equal(t, catalog.Open, (<-page).State, "the page hears it opened")

	clock.Advance(time.Hour)
	_, err = close_flag_usecase.New(clock, db, store, cpoutbox.New(db)).
		Execute(t.Context(), close_flag_usecase.In{Flag: out.ID.String(), Note: "Broke the rules."})
	require.NoError(t, err)
	require.Eventually(t, func() bool { return len(planet.all()) == 2 }, 10*time.Second, 20*time.Millisecond)
	assert.Equal(t, catalog.Closed, (<-page).State, "the page hears it closed")

	_, err = rename_flag_usecase.New(clock, db, store, cpoutbox.New(db)).
		Execute(t.Context(), rename_flag_usecase.In{Flag: out.ID.String(), Name: "Roazhon"})
	require.NoError(t, err)
	assert.Equal(t, catalog.Name("Roazhon"), (<-page).Name, "the page hears it renamed")

	assert.Equal(t, []string{"opened " + out.ID.String() + " Rennes", "closed " + out.ID.String()}, planet.all())
}

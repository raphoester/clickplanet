package flags_test

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/postgres_flag_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/keep_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/commands/keep_flag_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis/cpredistest"
)

// Planet sends it the first time a player plays for a flag, as its own command stream carries it.
func TestAFlagPlanetSaysWasPlayedIsNeverDeleted(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "flags", migrations.FS)
	redis := cpredistest.StartServer(t).Config()
	streams := cpredis.New(redis, prometheus.NewRegistry())
	require.NoError(t, streams.ConnectCtx(t.Context()))
	t.Cleanup(func() { _ = streams.Close() })

	store := postgres_flag_store.New(db)
	openedAt := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	require.NoError(t, store.Save(t.Context(), catalog.OpenFlag("r-0000000001", "Rennes", openedAt)))
	kept, err := cpredis.Handle(t.Context(), streams, keep_flag_handler.New(keep_flag_usecase.New(db, store)))
	require.NoError(t, err)
	running(t, kept)

	require.NoError(t, streams.Commands().Send(t.Context(),
		&flagsv1.KeepFlag{Flag: "r-0000000001", PlayedAt: timestamppb.New(openedAt.Add(time.Minute))}))

	require.Eventually(t, func() bool {
		flag, err := store.Flag(t.Context(), "r-0000000001")
		return err == nil && flag.TakeSnapshot().Played
	}, 10*time.Second, 20*time.Millisecond)
	flag, err := store.Flag(t.Context(), "r-0000000001")
	require.NoError(t, err)
	closed, err := flag.Closed(openedAt.Add(time.Hour))
	require.NoError(t, err)
	_, err = closed.Deleted(openedAt.Add(2 * time.Hour))
	assert.ErrorIs(t, err, catalog.ErrPlayed)
}

package relay_fortifications_usecase_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/relay_fortifications_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var at = time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)

func TestEachFortifyOnTheMapIsToldToTheOtherModules(t *testing.T) {
	tiles := inmemory_tile_storage.New(clicks.BordersOf(100, []uint32{20, 21}), inmemory_tile_storage.Config{},
		inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}), slog.New(slog.DiscardHandler))
	events := cpbootstrap.NewRecordedEvents()
	relay := relay_fortifications_usecase.New(tiles, events, cptime.NewFixedClock(at), slog.New(slog.DiscardHandler))

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go relay.Run(ctx)

	flags := []string{"fr", "de"}
	round := 0
	require.Eventually(t, func() bool {
		flag := flags[round%2]
		round++
		require.NoError(t, tiles.Set(ctx, 7, flag))
		require.NoError(t, tiles.Set(ctx, 20, flag))
		require.NoError(t, tiles.Set(ctx, 21, flag))
		_, err := tiles.Fortify(ctx, 21, flag, 10)
		require.NoError(t, err)
		return len(events.Published()) > 0
	}, time.Second, 20*time.Millisecond, "the relay hears a fortify once it follows the map")

	published := events.Published()
	last := published[len(published)-1].(*planetv1.Fortified) //nolint:forcetypeassert // the relay publishes nothing else.
	assert.Contains(t, flags, last.GetCountry())
	assert.True(t, proto.Equal(&planetv1.Fortified{
		Country: last.GetCountry(), Ground: "l1", LandmassId: 1, Tiles: 2, FortifiedAt: timestamppb.New(at),
	}, last))
}

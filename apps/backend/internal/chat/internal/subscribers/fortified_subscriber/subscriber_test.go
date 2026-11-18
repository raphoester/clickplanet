package fortified_subscriber_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/inmemory_announcement_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed/inprocess_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/fortified_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpboards"
)

var at = time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)

func subscriber(store *inmemory_announcement_storage.Storage) fortified_subscriber.Subscriber {
	return fortified_subscriber.New(announce_usecase.New(store, inprocess_feed.New(0, slog.New(slog.DiscardHandler))))
}

func TestAFortifyOfASizeableProvinceIsAnnouncedWithItsProvince(t *testing.T) {
	store := inmemory_announcement_storage.New()

	err := subscriber(store).Handle(t.Context(), &planetv1.Fortified{
		Country: "es", Ground: "fr", ProvinceId: 1031, LandmassId: 359, Tiles: 137, FortifiedAt: timestamppb.New(at),
	})

	require.NoError(t, err)
	kept := store.Kept()
	require.Len(t, kept, 1)
	assert.Equal(t, announcements.KindFortify, kept[0].Kind())
	assert.Equal(t, at, kept[0].At())
	assert.JSONEq(t, `{"country":"es","ground":"fr","province":1031,"landmass":359,"tiles":137}`, string(kept[0].Payload()),
		"the page names the province, or its landmass when it has no name of its own")
}

func TestAFortifyMadeBeforeProvincesIsAnnouncedWithItsLandmass(t *testing.T) {
	store := inmemory_announcement_storage.New()

	err := subscriber(store).Handle(t.Context(), &planetv1.Fortified{
		Country: "es", Ground: "fr", LandmassId: 359, Tiles: 137, FortifiedAt: timestamppb.New(at),
	})

	require.NoError(t, err)
	kept := store.Kept()
	require.Len(t, kept, 1)
	assert.JSONEq(t, `{"country":"es","ground":"fr","landmass":359,"tiles":137}`, string(kept[0].Payload()))
}

func TestAFortifyOfASmallProvinceIsNotAnnounced(t *testing.T) {
	store := inmemory_announcement_storage.New()

	err := subscriber(store).Handle(t.Context(), &planetv1.Fortified{
		Country: "bg", ProvinceId: 257, Tiles: 16, FortifiedAt: timestamppb.New(at),
	})

	require.NoError(t, err)
	assert.Empty(t, store.Kept())
}

func TestAFortifyWithNoTimeIsRefused(t *testing.T) {
	store := inmemory_announcement_storage.New()

	require.Error(t, subscriber(store).Handle(t.Context(), &planetv1.Fortified{Country: "es", Tiles: 137}))
	assert.Empty(t, store.Kept())
}

func TestAFortifyIsAsNewsworthyOnABoardOfItsOwnAsTheGroundItCovers(t *testing.T) {
	for tiles, announced := range map[uint32]bool{0: false, 1: true} {
		store := inmemory_announcement_storage.New()

		err := subscriber(store).Handle(t.Context(), &planetv1.Fortified{
			Country: "jp", ProvinceId: 12, Tiles: tiles, FortifiedAt: timestamppb.New(at), Board: cpboards.Fixture,
		})

		require.NoError(t, err)
		assert.Equal(t, announced, len(store.Kept()) == 1, "%d tiles of the fixture: 50 of earth-1 are one", tiles)
	}

	store := inmemory_announcement_storage.New()
	require.NoError(t, subscriber(store).Handle(t.Context(), &planetv1.Fortified{
		Country: "bg", ProvinceId: 257, Tiles: 49, FortifiedAt: timestamppb.New(at), Board: "earth-1",
	}))
	assert.Empty(t, store.Kept(), "50 on earth-1, as before")
}

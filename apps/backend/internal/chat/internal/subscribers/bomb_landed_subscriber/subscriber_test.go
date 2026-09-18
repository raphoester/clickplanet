package bomb_landed_subscriber_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/bomb_landed_subscriber"
)

var at = time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)

func subscriber(store *inmemory_announcement_storage.Storage) bomb_landed_subscriber.Subscriber {
	return bomb_landed_subscriber.New(announce_usecase.New(store, inprocess_feed.New(0, slog.New(slog.DiscardHandler))))
}

func TestABombIsAnnouncedAtTheTimeItLanded(t *testing.T) {
	store := inmemory_announcement_storage.New()

	err := subscriber(store).Handle(t.Context(), &planetv1.BombLanded{
		Country: "fr", TileId: 42, Ground: "de", Cleared: 3, LandedAt: timestamppb.New(at),
	})

	require.NoError(t, err)
	kept, err := store.Recent(t.Context(), time.Time{}, 10)
	require.NoError(t, err)
	require.Len(t, kept, 1)
	assert.Equal(t, announcements.KindBomb, kept[0].Kind)
	assert.Equal(t, at, kept[0].At)
	assert.JSONEq(t, `{"country":"fr","ground":"de","tile":42,"cleared":3}`, string(kept[0].Payload))
}

func TestABombInTheSeaHasNoGroundAndNoTile(t *testing.T) {
	store := inmemory_announcement_storage.New()

	err := subscriber(store).Handle(t.Context(), &planetv1.BombLanded{Country: "fr", LandedAt: timestamppb.New(at)})

	require.NoError(t, err)
	kept, err := store.Recent(t.Context(), time.Time{}, 10)
	require.NoError(t, err)
	require.Len(t, kept, 1)
	assert.JSONEq(t, `{"country":"fr","cleared":0}`, string(kept[0].Payload))
}

func TestABombWithNoTimeIsRefused(t *testing.T) {
	store := inmemory_announcement_storage.New()

	require.Error(t, subscriber(store).Handle(t.Context(), &planetv1.BombLanded{Country: "fr"}))

	kept, err := store.Recent(t.Context(), time.Time{}, 10)
	require.NoError(t, err)
	assert.Empty(t, kept)
}

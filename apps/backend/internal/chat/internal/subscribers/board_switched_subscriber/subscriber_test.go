package board_switched_subscriber_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/board_switched_subscriber"
)

var at = time.Date(2026, 11, 22, 4, 0, 0, 0, time.UTC)

func subscriber(store *inmemory_announcement_storage.Storage) board_switched_subscriber.Subscriber {
	return board_switched_subscriber.New(announce_usecase.New(store, inprocess_feed.New(0, slog.New(slog.DiscardHandler))))
}

func switched() *planetv1.BoardSwitched {
	return &planetv1.BoardSwitched{
		Map: "world", From: "earth-1", To: "earth-2", FromTiles: 260503, ToTiles: 1004682, SwitchedAt: timestamppb.New(at),
		Position: 41,
	}
}

func TestTheWorldsSwitchIsAnnouncedInTheWorldChat(t *testing.T) {
	store := inmemory_announcement_storage.New()

	require.NoError(t, subscriber(store).Handle(t.Context(), switched()))

	kept := store.Kept()
	require.Len(t, kept, 1)
	assert.Equal(t, announcements.KindBoardSwitched, kept[0].Kind())
	assert.Equal(t, at, kept[0].At())
	assert.Equal(t, messages.World, kept[0].Room())
	assert.JSONEq(t, `{"from":"earth-1","to":"earth-2","fromTiles":260503,"toTiles":1004682}`, string(kept[0].Payload()))
}

func TestASwitchToldTwiceIsAnnouncedOnce(t *testing.T) {
	store := inmemory_announcement_storage.New()

	require.NoError(t, subscriber(store).Handle(t.Context(), switched()))
	require.NoError(t, subscriber(store).Handle(t.Context(), switched()))

	assert.Len(t, store.Kept(), 1)
}

func TestASwitchWithNoTimeIsRefusedAndAnotherMapsIsNotAnnounced(t *testing.T) {
	store := inmemory_announcement_storage.New()
	event := switched()
	event.SwitchedAt = nil
	require.Error(t, subscriber(store).Handle(t.Context(), event))

	event = switched()
	event.Map = "battle"
	require.NoError(t, subscriber(store).Handle(t.Context(), event))

	assert.Empty(t, store.Kept())
}

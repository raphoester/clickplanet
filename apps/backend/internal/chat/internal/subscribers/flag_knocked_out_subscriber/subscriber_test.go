package flag_knocked_out_subscriber_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/flag_knocked_out_subscriber"
)

var at = time.Date(2026, 11, 11, 20, 4, 0, 0, time.UTC)

func subscriber(store *inmemory_announcement_storage.Storage) flag_knocked_out_subscriber.Subscriber {
	return flag_knocked_out_subscriber.New(announce_usecase.New(store, inprocess_feed.New(0, slog.New(slog.DiscardHandler))))
}

func TestEveryKnockOutIsAnnouncedInTheWorld(t *testing.T) {
	store := inmemory_announcement_storage.New()

	err := subscriber(store).Handle(t.Context(), &planetv1.FlagKnockedOut{
		Country: "fr", By: "de", TileId: 4, Cleared: 0, KnockedOutAt: timestamppb.New(at), Position: 9,
	})

	require.NoError(t, err)
	kept := store.Kept()
	require.Len(t, kept, 1, "an empty capital too")
	assert.Equal(t, announcements.KindKnockOut, kept[0].Kind())
	assert.Equal(t, at, kept[0].At())
	assert.Equal(t, messages.World, kept[0].Room())
	assert.JSONEq(t, `{"country":"fr","by":"de"}`, string(kept[0].Payload()))
}

func TestAKnockOutToldTwiceIsAnnouncedOnce(t *testing.T) {
	store := inmemory_announcement_storage.New()
	event := &planetv1.FlagKnockedOut{Country: "fr", By: "de", KnockedOutAt: timestamppb.New(at), Position: 9}

	require.NoError(t, subscriber(store).Handle(t.Context(), event))
	require.NoError(t, subscriber(store).Handle(t.Context(), event))

	assert.Len(t, store.Kept(), 1)
}

func TestAKnockOutWithNoTimeIsRefused(t *testing.T) {
	store := inmemory_announcement_storage.New()

	require.Error(t, subscriber(store).Handle(t.Context(), &planetv1.FlagKnockedOut{Country: "fr", By: "de"}))
	assert.Empty(t, store.Kept())
}

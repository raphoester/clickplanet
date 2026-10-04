package lead_changed_subscriber_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/inmemory_announcement_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed/inprocess_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/lead_changed_subscriber"
)

var at = time.Date(2026, 10, 31, 21, 42, 0, 0, time.UTC)

func subscriber(store *inmemory_announcement_storage.Storage) lead_changed_subscriber.Subscriber {
	return lead_changed_subscriber.New(announce_usecase.New(store, inprocess_feed.New(0, slog.New(slog.DiscardHandler))))
}

func TestALeadChangeIsAnnouncedAtTheTimeItChanged(t *testing.T) {
	store := inmemory_announcement_storage.New()

	err := subscriber(store).Handle(t.Context(), &seasonsv1.LeadChanged{
		Season: 0, Leader: "bg", Passed: "fr", ChangedAt: timestamppb.New(at),
	})

	require.NoError(t, err)
	kept := store.Kept()
	require.Len(t, kept, 1)
	assert.Equal(t, announcements.KindLeadChanged, kept[0].Kind())
	assert.Equal(t, at, kept[0].At())
	assert.JSONEq(t, `{"season":0,"leader":"bg","passed":"fr"}`, string(kept[0].Payload()))
}

func TestALeadChangeWithNoTimeOrNoCountryIsRefused(t *testing.T) {
	store := inmemory_announcement_storage.New()

	require.Error(t, subscriber(store).Handle(t.Context(), &seasonsv1.LeadChanged{Leader: "bg", Passed: "fr"}))
	require.Error(t, subscriber(store).Handle(t.Context(), &seasonsv1.LeadChanged{Leader: "bg", ChangedAt: timestamppb.New(at)}))
	require.Error(t, subscriber(store).Handle(t.Context(), &seasonsv1.LeadChanged{Passed: "fr", ChangedAt: timestamppb.New(at)}))

	assert.Empty(t, store.Kept())
}

package season_ended_subscriber_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/season_ended_subscriber"
)

var ended = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)

func subscriber(store *inmemory_announcement_storage.Storage) season_ended_subscriber.Subscriber {
	return season_ended_subscriber.New(announce_usecase.New(store, inprocess_feed.New(0, slog.New(slog.DiscardHandler)), &announcements.SequentialIDs{}))
}

func TestTheWinnerIsAnnouncedAtTheEnd(t *testing.T) {
	store := inmemory_announcement_storage.New()

	err := subscriber(store).Handle(t.Context(), &seasonsv1.SeasonEnded{Season: 0, Winner: "dz", EndedAt: timestamppb.New(ended)})

	require.NoError(t, err)
	kept := store.Kept()
	require.Len(t, kept, 1)
	assert.Equal(t, announcements.KindSeasonWon, kept[0].Kind())
	assert.Equal(t, ended, kept[0].At())
	assert.JSONEq(t, `{"season":0,"winner":"dz"}`, string(kept[0].Payload()))
}

func TestAnEndWithNoTimeOrNoWinnerIsRefused(t *testing.T) {
	store := inmemory_announcement_storage.New()

	require.Error(t, subscriber(store).Handle(t.Context(), &seasonsv1.SeasonEnded{Winner: "dz"}))
	require.Error(t, subscriber(store).Handle(t.Context(), &seasonsv1.SeasonEnded{EndedAt: timestamppb.New(ended)}))

	assert.Empty(t, store.Kept())
}

func TestASeasonIsWonOnceHoweverOftenItsEndIsTold(t *testing.T) {
	store := inmemory_announcement_storage.New()
	event := &seasonsv1.SeasonEnded{Season: 0, Winner: "dz", EndedAt: timestamppb.New(ended)}

	require.NoError(t, subscriber(store).Handle(t.Context(), event))
	require.NoError(t, subscriber(store).Handle(t.Context(), event), "a restart after the end tells it again")

	assert.Len(t, store.Kept(), 1)
}

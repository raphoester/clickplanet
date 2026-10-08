package round_closed_subscriber_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/round_closed_subscriber"
)

var endedAt = time.Date(2026, 10, 16, 21, 0, 0, 0, time.UTC)

func subscriber(store *inmemory_announcement_storage.Storage) round_closed_subscriber.Subscriber {
	return round_closed_subscriber.New(announce_usecase.New(store, inprocess_feed.New(0, slog.New(slog.DiscardHandler)), &announcements.SequentialIDs{}))
}

func TestARoundIsAnnouncedWithItsPodiumAtTheTimeItEnded(t *testing.T) {
	store := inmemory_announcement_storage.New()

	err := subscriber(store).Handle(t.Context(), &seasonsv1.RoundClosed{
		Number:  5,
		EndedAt: timestamppb.New(endedAt),
		Results: []*seasonsv1.RoundResult{
			{Country: "fr", Rank: 1, Points: 25},
			{Country: "de", Rank: 2, Points: 18},
			{Country: "es", Rank: 3, Points: 15},
			{Country: "it", Rank: 4, Points: 12},
		},
	})

	require.NoError(t, err)
	kept := store.Kept()
	require.Len(t, kept, 1)
	assert.Equal(t, announcements.KindRound, kept[0].Kind())
	assert.Equal(t, endedAt, kept[0].At())
	assert.JSONEq(t, `{"number":5,"podium":[
		{"country":"fr","rank":1,"points":25},
		{"country":"de","rank":2,"points":18},
		{"country":"es","rank":3,"points":15}
	]}`, string(kept[0].Payload()))
}

func TestTheFinaleIsAnnouncedAsTheFinale(t *testing.T) {
	store := inmemory_announcement_storage.New()

	err := subscriber(store).Handle(t.Context(), &seasonsv1.RoundClosed{
		Number:  23,
		Finale:  true,
		EndedAt: timestamppb.New(endedAt),
		Results: []*seasonsv1.RoundResult{{Country: "fr", Rank: 1, Points: 75}},
	})

	require.NoError(t, err)
	kept := store.Kept()
	require.Len(t, kept, 1)
	assert.JSONEq(t, `{"number":23,"finale":true,"podium":[{"country":"fr","rank":1,"points":75}]}`, string(kept[0].Payload()))
}

func TestARoundWithNoEndIsRefused(t *testing.T) {
	store := inmemory_announcement_storage.New()

	require.Error(t, subscriber(store).Handle(t.Context(), &seasonsv1.RoundClosed{Number: 1}))

	assert.Empty(t, store.Kept())
}

package round_closed_subscriber_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/inmemory_announcement_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed/inprocess_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/round_closed_subscriber"
)

var endedAt = time.Date(2026, 10, 16, 21, 0, 0, 0, time.UTC)

func subscriber(store *inmemory_announcement_storage.Storage) round_closed_subscriber.Subscriber {
	return round_closed_subscriber.New(announce_usecase.New(store, inprocess_feed.New(0, slog.New(slog.DiscardHandler))))
}

func TestARoundIsAnnouncedWithItsPodiumAtTheTimeItEnded(t *testing.T) {
	store := inmemory_announcement_storage.New()

	err := subscriber(store).Handle(t.Context(), &scoringv1.RoundClosed{
		Number:  5,
		EndedAt: timestamppb.New(endedAt),
		Results: []*scoringv1.RoundResult{
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

	err := subscriber(store).Handle(t.Context(), &scoringv1.RoundClosed{
		Number:  23,
		Finale:  true,
		EndedAt: timestamppb.New(endedAt),
		Results: []*scoringv1.RoundResult{{Country: "fr", Rank: 1, Points: 75}},
	})

	require.NoError(t, err)
	kept := store.Kept()
	require.Len(t, kept, 1)
	assert.JSONEq(t, `{"number":23,"finale":true,"podium":[{"country":"fr","rank":1,"points":75}]}`, string(kept[0].Payload()))
}

func TestARoundOfItsOwnIsAnnouncedAsOneApartFromTheDayOfItsNumber(t *testing.T) {
	store := inmemory_announcement_storage.New()
	day := &scoringv1.RoundClosed{Number: 1, EndedAt: timestamppb.New(endedAt), Times: 1}
	battle := &scoringv1.RoundClosed{
		Number: 1, Finale: true, Own: true, Times: 3, EndedAt: timestamppb.New(endedAt),
		Results: []*scoringv1.RoundResult{{Country: "fr", Rank: 1, Points: 75}},
	}

	require.NoError(t, subscriber(store).Handle(t.Context(), day))
	require.NoError(t, subscriber(store).Handle(t.Context(), battle))

	kept := store.Kept()
	require.Len(t, kept, 2, "day 1 and the first round of its own")
	assert.JSONEq(t, `{"number":1,"finale":true,"own":true,"times":3,"podium":[{"country":"fr","rank":1,"points":75}]}`,
		string(kept[1].Payload()))
}

func TestARoundToldTwiceIsAnnouncedOnce(t *testing.T) {
	store := inmemory_announcement_storage.New()
	closed := func(season, number uint32) *scoringv1.RoundClosed {
		return &scoringv1.RoundClosed{Season: season, Number: number, EndedAt: timestamppb.New(endedAt)}
	}

	require.NoError(t, subscriber(store).Handle(t.Context(), closed(0, 5)))
	require.NoError(t, subscriber(store).Handle(t.Context(), closed(0, 5)))
	require.NoError(t, subscriber(store).Handle(t.Context(), closed(1, 5)))

	assert.Len(t, store.Kept(), 2, "round 5 of seasons 0 and 1")
}

func TestARoundWithNoEndIsRefused(t *testing.T) {
	store := inmemory_announcement_storage.New()

	require.Error(t, subscriber(store).Handle(t.Context(), &scoringv1.RoundClosed{Number: 1}))

	assert.Empty(t, store.Kept())
}

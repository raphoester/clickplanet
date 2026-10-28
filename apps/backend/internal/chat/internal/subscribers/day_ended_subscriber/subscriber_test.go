package day_ended_subscriber_test

import (
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/inmemory_announcement_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed/inprocess_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/day_ended_subscriber"
)

var endedAt = time.Date(2026, 10, 16, 21, 0, 0, 0, time.UTC)

var scored = announcements.FakeResults{Rounds: map[announcements.RoundID][]announcements.Place{
	"season-0/day-5": {
		announcements.PlaceOf("fr", 1, 25),
		announcements.PlaceOf("de", 2, 18),
		announcements.PlaceOf("es", 3, 15),
		announcements.PlaceOf("it", 4, 12),
	},
	"season-1/day-5": {announcements.PlaceOf("fr", 1, 25)},
}}

func subscriber(store *inmemory_announcement_storage.Storage, results announcements.FakeResults) day_ended_subscriber.Subscriber {
	return day_ended_subscriber.New(announce_round_usecase.New(results,
		announce_usecase.New(store, inprocess_feed.New(0, slog.New(slog.DiscardHandler)))))
}

func dayEnded(season, day uint32) *gamemasterv1.DayEnded {
	return &gamemasterv1.DayEnded{
		Season: season, Day: day, RoundId: fmt.Sprintf("season-%d/day-%d", season, day), EndedAt: timestamppb.New(endedAt),
	}
}

func TestADayIsAnnouncedWithItsPodiumAtTheTimeItEnded(t *testing.T) {
	store := inmemory_announcement_storage.New()

	require.NoError(t, subscriber(store, scored).Handle(t.Context(), dayEnded(0, 5)))

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

func TestADayToldTwiceIsAnnouncedOnce(t *testing.T) {
	store := inmemory_announcement_storage.New()

	require.NoError(t, subscriber(store, scored).Handle(t.Context(), dayEnded(0, 5)))
	require.NoError(t, subscriber(store, scored).Handle(t.Context(), dayEnded(0, 5)))
	require.NoError(t, subscriber(store, scored).Handle(t.Context(), dayEnded(1, 5)))

	assert.Len(t, store.Kept(), 2, "day 5 of seasons 0 and 1")
}

func TestADayScoringCannotAnswerForIsNotAnnounced(t *testing.T) {
	store := inmemory_announcement_storage.New()

	err := subscriber(store, announcements.FakeResults{Err: errors.New("scoring is down")}).Handle(t.Context(), dayEnded(0, 5))

	require.Error(t, err)
	assert.Empty(t, store.Kept())
}

func TestADayWithNoEndIsRefused(t *testing.T) {
	store := inmemory_announcement_storage.New()

	require.Error(t, subscriber(store, scored).Handle(t.Context(), &gamemasterv1.DayEnded{Day: 5, RoundId: "season-0/day-5"}))

	assert.Empty(t, store.Kept())
}

func TestADayWithNoRoundIsRefused(t *testing.T) {
	store := inmemory_announcement_storage.New()

	err := subscriber(store, scored).Handle(t.Context(), &gamemasterv1.DayEnded{Day: 5, EndedAt: timestamppb.New(endedAt)})

	require.ErrorIs(t, err, announcements.ErrNoRound)
	assert.Empty(t, store.Kept())
}

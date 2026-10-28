package battle_ended_subscriber_test

import (
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
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/battle_ended_subscriber"
)

var endedAt = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)

const (
	finaleRound = "0b4f5c0e-6a52-4f39-9d3e-3c1e8f0a7d21"
	firstRound  = "5d3a1c2e-1b7f-4e0c-8a9d-2f6b4c8e0a13"
)

var scored = announcements.FakeResults{Rounds: map[announcements.RoundID][]announcements.Place{
	finaleRound: {announcements.PlaceOf("fr", 1, 75), announcements.PlaceOf("de", 2, 54)},
	firstRound:  {announcements.PlaceOf("de", 1, 50)},
}}

func subscriber(store *inmemory_announcement_storage.Storage) battle_ended_subscriber.Subscriber {
	return battle_ended_subscriber.New(announce_round_usecase.New(scored,
		announce_usecase.New(store, inprocess_feed.New(0, slog.New(slog.DiscardHandler)))))
}

func TestTheFinaleIsAnnouncedAsARoundOfItsOwnAtItsTimes(t *testing.T) {
	store := inmemory_announcement_storage.New()

	err := subscriber(store).Handle(t.Context(), &gamemasterv1.BattleEnded{
		Season: 0, Number: 2, Times: 3, EndsSeason: true, RoundId: finaleRound, EndedAt: timestamppb.New(endedAt),
	})

	require.NoError(t, err)
	kept := store.Kept()
	require.Len(t, kept, 1)
	assert.Equal(t, announcements.KindRound, kept[0].Kind())
	assert.Equal(t, endedAt, kept[0].At())
	assert.JSONEq(t, `{"number":2,"finale":true,"own":true,"times":3,"podium":[
		{"country":"fr","rank":1,"points":75},
		{"country":"de","rank":2,"points":54}
	]}`, string(kept[0].Payload()))
}

func TestABattleIsAnnouncedApartFromTheDayOfItsNumber(t *testing.T) {
	store := inmemory_announcement_storage.New()
	battle := &gamemasterv1.BattleEnded{Number: 1, Times: 2, RoundId: firstRound, EndedAt: timestamppb.New(endedAt)}

	require.NoError(t, subscriber(store).Handle(t.Context(), battle))
	require.NoError(t, subscriber(store).Handle(t.Context(), battle))
	day, err := announcements.DayOf(0, 1).Announcement(endedAt)
	require.NoError(t, err)

	kept := store.Kept()
	require.Len(t, kept, 1, "a battle told twice is announced once")
	assert.NotEqual(t, day.ID(), kept[0].ID())
}

func TestABattleWithNoEndIsRefused(t *testing.T) {
	store := inmemory_announcement_storage.New()

	require.Error(t, subscriber(store).Handle(t.Context(), &gamemasterv1.BattleEnded{Number: 1, RoundId: firstRound}))

	assert.Empty(t, store.Kept())
}

func TestABattleWithNoRoundIsRefused(t *testing.T) {
	store := inmemory_announcement_storage.New()

	err := subscriber(store).Handle(t.Context(), &gamemasterv1.BattleEnded{Number: 1, EndedAt: timestamppb.New(endedAt)})

	require.ErrorIs(t, err, announcements.ErrNoRound)
	assert.Empty(t, store.Kept())
}

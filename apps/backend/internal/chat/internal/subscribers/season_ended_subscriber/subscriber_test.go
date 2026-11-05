package season_ended_subscriber_test

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/inmemory_announcement_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_champion_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed/inprocess_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/season_ended_subscriber"
)

var endedAt = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)

var scored = announcements.FakeResults{Tables: map[announcements.BookID][]announcements.Place{
	"season-0": {announcements.PlaceOf("de", 1, 118), announcements.PlaceOf("fr", 2, 97)},
	"season-1": nil,
}}

func subscriber(store *inmemory_announcement_storage.Storage, results announcements.FakeResults) season_ended_subscriber.Subscriber {
	return season_ended_subscriber.New(announce_champion_usecase.New(results,
		announce_usecase.New(store, inprocess_feed.New(0, slog.New(slog.DiscardHandler)))))
}

var seasonZero = &gamemasterv1.SeasonEnded{Season: 0, BookId: "season-0", EndedAt: timestamppb.New(endedAt)}

func TestTheSeasonsChampionIsAnnouncedAtTheTimeItEnded(t *testing.T) {
	store := inmemory_announcement_storage.New()

	require.NoError(t, subscriber(store, scored).Handle(t.Context(), seasonZero))

	kept := store.Kept()
	require.Len(t, kept, 1)
	assert.Equal(t, announcements.KindChampion, kept[0].Kind())
	assert.Equal(t, endedAt, kept[0].At())
	assert.JSONEq(t, `{"season":0,"countries":["de"]}`, string(kept[0].Payload()))
}

func TestASeasonToldAgainIsNotAnnouncedAgain(t *testing.T) {
	store := inmemory_announcement_storage.New()

	require.NoError(t, subscriber(store, scored).Handle(t.Context(), seasonZero))
	require.NoError(t, subscriber(store, scored).Handle(t.Context(), seasonZero))

	assert.Len(t, store.Kept(), 1)
}

func TestASeasonNobodyScoredInIsNotAnnounced(t *testing.T) {
	store := inmemory_announcement_storage.New()

	err := subscriber(store, scored).Handle(t.Context(),
		&gamemasterv1.SeasonEnded{Season: 1, BookId: "season-1", EndedAt: timestamppb.New(endedAt)})

	require.NoError(t, err)
	assert.Empty(t, store.Kept())
}

func TestASeasonScoringCannotAnswerForIsNotAnnounced(t *testing.T) {
	store := inmemory_announcement_storage.New()

	err := subscriber(store, announcements.FakeResults{Err: errors.New("scoring is down")}).Handle(t.Context(), seasonZero)

	require.Error(t, err)
	assert.Empty(t, store.Kept())
}

func TestASeasonWithNoEndIsRefused(t *testing.T) {
	store := inmemory_announcement_storage.New()

	require.Error(t, subscriber(store, scored).Handle(t.Context(), &gamemasterv1.SeasonEnded{BookId: "season-0"}))

	assert.Empty(t, store.Kept())
}

func TestASeasonWithNoBookIsRefused(t *testing.T) {
	store := inmemory_announcement_storage.New()

	err := subscriber(store, scored).Handle(t.Context(), &gamemasterv1.SeasonEnded{EndedAt: timestamppb.New(endedAt)})

	require.ErrorIs(t, err, announcements.ErrNoBook)
	assert.Empty(t, store.Kept())
}

func TestASeasonEndedByAFinaleBattleAnnouncesTheChampionAloneAndNotTheBattleAgain(t *testing.T) {
	store := inmemory_announcement_storage.New()
	ended := &gamemasterv1.SeasonEnded{
		Season: 0, BookId: "season-0", EndedAt: timestamppb.New(endedAt), FinaleRoundId: "b",
		FinaleBattle: &gamemasterv1.BattleEnded{Season: 0, Number: 3, Times: 3, EndsSeason: true, RoundId: "b", EndedAt: timestamppb.New(endedAt)},
	}

	require.NoError(t, subscriber(store, scored).Handle(t.Context(), ended))

	kept := store.Kept()
	require.Len(t, kept, 1)
	assert.Equal(t, announcements.KindChampion, kept[0].Kind())
}

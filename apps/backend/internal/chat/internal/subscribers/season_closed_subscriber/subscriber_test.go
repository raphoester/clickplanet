package season_closed_subscriber_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/season_closed_subscriber"
)

var endedAt = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)

func subscriber(store *inmemory_announcement_storage.Storage) season_closed_subscriber.Subscriber {
	return season_closed_subscriber.New(announce_usecase.New(store, inprocess_feed.New(0, slog.New(slog.DiscardHandler))))
}

var seasonZero = &scoringv1.SeasonClosed{
	Season:  0,
	EndedAt: timestamppb.New(endedAt),
	Table: []*scoringv1.SeasonPlace{
		{Country: "de", Rank: 1, Points: 118, RoundsWon: 2, FinalePoints: 75},
		{Country: "fr", Rank: 2, Points: 97, RoundsWon: 1, FinalePoints: 54},
	},
}

func TestTheSeasonsChampionIsAnnouncedAtTheTimeItEnded(t *testing.T) {
	store := inmemory_announcement_storage.New()

	require.NoError(t, subscriber(store).Handle(t.Context(), seasonZero))

	kept := store.Kept()
	require.Len(t, kept, 1)
	assert.Equal(t, announcements.KindChampion, kept[0].Kind())
	assert.Equal(t, endedAt, kept[0].At())
	assert.JSONEq(t, `{"season":0,"countries":["de"]}`, string(kept[0].Payload()))
}

func TestASeasonToldAgainIsNotAnnouncedAgain(t *testing.T) {
	store := inmemory_announcement_storage.New()

	require.NoError(t, subscriber(store).Handle(t.Context(), seasonZero))
	require.NoError(t, subscriber(store).Handle(t.Context(), seasonZero))

	assert.Len(t, store.Kept(), 1)
}

func TestASeasonNobodyScoredInIsNotAnnounced(t *testing.T) {
	store := inmemory_announcement_storage.New()

	require.NoError(t, subscriber(store).Handle(t.Context(), &scoringv1.SeasonClosed{EndedAt: timestamppb.New(endedAt)}))

	assert.Empty(t, store.Kept())
}

func TestASeasonWithNoEndIsRefused(t *testing.T) {
	store := inmemory_announcement_storage.New()

	require.Error(t, subscriber(store).Handle(t.Context(), &scoringv1.SeasonClosed{Table: seasonZero.GetTable()}))

	assert.Empty(t, store.Kept())
}

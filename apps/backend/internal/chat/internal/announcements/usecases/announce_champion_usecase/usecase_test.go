package announce_champion_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/inmemory_announcement_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_champion_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
)

var at = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)

type recordedFeed struct{ updates []feed.Update }

func (r *recordedFeed) Publish(update feed.Update) { r.updates = append(r.updates, update) }

type failingAppender struct{}

func (failingAppender) AppendOnce(context.Context, announcements.Announcement) (bool, error) {
	return false, errors.New("postgres is down")
}

func champion(country string) announcements.Champion {
	return announcements.ChampionOf(0, []announcements.Place{announcements.PlaceOf(country, 1, 118)})
}

func TestTheChampionIsKeptThenPublished(t *testing.T) {
	store := inmemory_announcement_storage.New()
	updates := &recordedFeed{}

	err := announce_champion_usecase.New(store, updates).Execute(t.Context(), announce_champion_usecase.In{Champion: champion("de"), At: at})

	require.NoError(t, err)
	kept := store.Kept()
	require.Len(t, kept, 1)
	assert.Equal(t, announcements.KindChampion, kept[0].Kind())
	assert.Equal(t, at, kept[0].At())
	assert.Equal(t, []feed.Update{feed.Announced(kept[0])}, updates.updates)
}

func TestASeasonToldTwiceIsAnnouncedOnce(t *testing.T) {
	store := inmemory_announcement_storage.New()
	updates := &recordedFeed{}
	useCase := announce_champion_usecase.New(store, updates)

	require.NoError(t, useCase.Execute(t.Context(), announce_champion_usecase.In{Champion: champion("de"), At: at}))
	require.NoError(t, useCase.Execute(t.Context(), announce_champion_usecase.In{Champion: champion("fr"), At: at.Add(time.Minute)}))

	kept := store.Kept()
	require.Len(t, kept, 1)
	assert.JSONEq(t, `{"season":0,"countries":["de"]}`, string(kept[0].Payload()))
	assert.Len(t, updates.updates, 1)
}

func TestAChampionThatCannotBeKeptIsNotPublished(t *testing.T) {
	updates := &recordedFeed{}

	err := announce_champion_usecase.New(failingAppender{}, updates).Execute(t.Context(),
		announce_champion_usecase.In{Champion: champion("de"), At: at})

	require.Error(t, err)
	assert.Empty(t, updates.updates)
}

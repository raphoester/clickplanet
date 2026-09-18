package announce_usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/inmemory_announcement_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
)

var at = time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)

type recordedFeed struct{ updates []feed.Update }

func (r *recordedFeed) Publish(update feed.Update) { r.updates = append(r.updates, update) }

type failingAppender struct{}

func (failingAppender) Append(context.Context, announcements.Announcement) error {
	return errors.New("postgres is down")
}

func TestAnAnnouncementIsKeptThenPublished(t *testing.T) {
	store := inmemory_announcement_storage.New()
	updates := &recordedFeed{}
	payload := json.RawMessage(`{"country":"fr","cleared":0}`)

	err := announce_usecase.New(store, updates).Execute(t.Context(),
		announce_usecase.In{Kind: announcements.KindBomb, At: at, Payload: payload})

	require.NoError(t, err)
	kept, err := store.Recent(t.Context(), time.Time{}, 10)
	require.NoError(t, err)
	require.Len(t, kept, 1)
	assert.NotEmpty(t, kept[0].ID)
	assert.Equal(t, announcements.KindBomb, kept[0].Kind)
	assert.Equal(t, at, kept[0].At)

	require.Len(t, updates.updates, 1)
	assert.Equal(t, kept[0], *updates.updates[0].Announcement)
}

func TestAnAnnouncementThatCannotBeKeptIsNotPublished(t *testing.T) {
	updates := &recordedFeed{}

	err := announce_usecase.New(failingAppender{}, updates).Execute(t.Context(),
		announce_usecase.In{Kind: announcements.KindBomb, At: at, Payload: json.RawMessage(`{}`)})

	require.Error(t, err)
	assert.Empty(t, updates.updates)
}

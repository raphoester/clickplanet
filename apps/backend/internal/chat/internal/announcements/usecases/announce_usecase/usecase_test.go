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

func (failingAppender) AppendOnce(context.Context, announcements.Announcement) (bool, error) {
	return false, errors.New("postgres is down")
}

func bomb(t *testing.T, position uint64, at time.Time) announcements.Announcement {
	t.Helper()

	announcement, err := announcements.BombOf(position, "fr", "de", 7, 42, 3).Announcement(at)
	require.NoError(t, err)
	return announcement
}

func TestAnAnnouncementIsKeptThenPublished(t *testing.T) {
	store := inmemory_announcement_storage.New()
	updates := &recordedFeed{}

	require.NoError(t, announce_usecase.New(store, updates).Execute(t.Context(), bomb(t, 7, at)))

	assert.Equal(t, []announcements.Announcement{bomb(t, 7, at)}, store.Kept())
	assert.Equal(t, []feed.Update{feed.Announced(store.Kept()[0])}, updates.updates)
}

func TestAFactToldTwiceIsAnnouncedAndPublishedOnce(t *testing.T) {
	store := inmemory_announcement_storage.New()
	updates := &recordedFeed{}
	useCase := announce_usecase.New(store, updates)

	require.NoError(t, useCase.Execute(t.Context(), bomb(t, 7, at)))
	require.NoError(t, useCase.Execute(t.Context(), bomb(t, 7, at.Add(time.Minute))))
	require.NoError(t, useCase.Execute(t.Context(), bomb(t, 8, at)))

	assert.Len(t, store.Kept(), 2, "bombs 7 and 8")
	assert.Len(t, updates.updates, 2)
}

func TestAnAnnouncementThatCannotBeKeptIsNotPublished(t *testing.T) {
	updates := &recordedFeed{}

	require.Error(t, announce_usecase.New(failingAppender{}, updates).Execute(t.Context(), bomb(t, 7, at)))

	assert.Empty(t, updates.updates)
}

func TestAnAnnouncementOfAKindNobodyKnowsIsRefusedAndNeitherKeptNorPublished(t *testing.T) {
	store := inmemory_announcement_storage.New()
	updates := &recordedFeed{}

	err := announce_usecase.New(store, updates).Execute(t.Context(),
		announcements.NewAnnouncement(announcements.AnnouncementID{15: 1}, "meteor", at, json.RawMessage(`{}`)))

	require.ErrorIs(t, err, announcements.ErrUnknownKind)
	assert.Empty(t, store.Kept())
	assert.Empty(t, updates.updates)
}

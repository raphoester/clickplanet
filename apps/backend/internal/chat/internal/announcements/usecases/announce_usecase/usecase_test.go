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
	kept := store.Kept()
	require.Len(t, kept, 1)
	assert.NotEmpty(t, kept[0].ID())
	assert.Equal(t, announcements.KindBomb, kept[0].Kind())
	assert.Equal(t, at, kept[0].At())

	require.Len(t, updates.updates, 1)
	published, announced := updates.updates[0].Announcement()
	require.True(t, announced)
	assert.Equal(t, kept[0], published)
}

func TestAnAnnouncementThatCannotBeKeptIsNotPublished(t *testing.T) {
	updates := &recordedFeed{}

	err := announce_usecase.New(failingAppender{}, updates).Execute(t.Context(),
		announce_usecase.In{Kind: announcements.KindBomb, At: at, Payload: json.RawMessage(`{}`)})

	require.Error(t, err)
	assert.Empty(t, updates.updates)
}

func TestAnAnnouncementOfAKindNobodyKnowsIsRefusedAndNeitherKeptNorPublished(t *testing.T) {
	store := inmemory_announcement_storage.New()
	updates := &recordedFeed{}

	err := announce_usecase.New(store, updates).Execute(t.Context(),
		announce_usecase.In{Kind: "meteor", At: at, Payload: json.RawMessage(`{}`)})

	require.ErrorIs(t, err, announcements.ErrUnknownKind)
	assert.Empty(t, store.Kept())
	assert.Empty(t, updates.updates)
}

func TestAnAnnouncementMadeOnceIsKeptAndShownOnce(t *testing.T) {
	store := inmemory_announcement_storage.New()
	updates := &recordedFeed{}
	useCase := announce_usecase.New(store, updates)
	in := announce_usecase.In{Kind: announcements.KindSeasonWon, At: at, Payload: json.RawMessage(`{"season":0,"winner":"dz"}`), Once: "season-0"}

	require.NoError(t, useCase.Execute(t.Context(), in))
	require.NoError(t, useCase.Execute(t.Context(), in), "a second telling is no fault")

	assert.Len(t, store.Kept(), 1)
	assert.Len(t, updates.updates, 1, "nobody is shown it twice")

	in.Once = "season-1"
	require.NoError(t, useCase.Execute(t.Context(), in))
	assert.Len(t, store.Kept(), 2)
}

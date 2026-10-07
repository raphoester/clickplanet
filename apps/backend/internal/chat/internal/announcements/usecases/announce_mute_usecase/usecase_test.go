package announce_mute_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/inmemory_announcement_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_mute_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

var (
	at    = time.Date(2026, 10, 7, 20, 0, 0, 0, time.UTC)
	bully = messages.AccountID{15: 1}
)

type recordedFeed struct{ updates []feed.Update }

func (r *recordedFeed) Publish(update feed.Update) { r.updates = append(r.updates, update) }

type fakeAuthors struct{ err error }

func (f fakeAuthors) Author(context.Context, messages.AccountID) (messages.Author, error) {
	if f.err != nil {
		return messages.Author{}, f.err
	}
	return messages.AuthorOf("guest_a1b2c3", false, 0, 0, messages.Title{}), nil
}

func TestAMuteIsAnnouncedUnderTheNameOfTheAccountAndForItsDuration(t *testing.T) {
	store := inmemory_announcement_storage.New()
	updates := &recordedFeed{}

	err := announce_mute_usecase.New(fakeAuthors{}, announce_usecase.New(store, updates)).Execute(t.Context(),
		announce_mute_usecase.In{Account: bully, At: at, Duration: 90 * time.Minute})

	require.NoError(t, err)
	kept := store.Kept()
	require.Len(t, kept, 1)
	assert.Equal(t, announcements.KindMute, kept[0].Kind())
	assert.Equal(t, at, kept[0].At())
	assert.JSONEq(t, `{"name":"guest_a1b2c3","seconds":5400}`, string(kept[0].Payload()))
	require.Len(t, updates.updates, 1)
	published, announced := updates.updates[0].Announcement()
	require.True(t, announced, "every open chat sees it at once")
	assert.Equal(t, kept[0], published)
}

func TestAMuteNobodyCanNameIsNotAnnounced(t *testing.T) {
	store := inmemory_announcement_storage.New()
	updates := &recordedFeed{}

	err := announce_mute_usecase.New(fakeAuthors{err: errors.New("the player module is down")}, announce_usecase.New(store, updates)).
		Execute(t.Context(), announce_mute_usecase.In{Account: bully, At: at, Duration: time.Hour})

	require.Error(t, err)
	assert.Empty(t, store.Kept())
	assert.Empty(t, updates.updates)
}

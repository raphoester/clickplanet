package announce_rally_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/inmemory_announcement_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_rally_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

var (
	at        = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	professor = messages.AccountID{15: 1}
)

type recordedFeed struct{ updates []feed.Update }

func (r *recordedFeed) Publish(update feed.Update) { r.updates = append(r.updates, update) }

type fakeAuthors struct{ err error }

func (f fakeAuthors) Author(context.Context, messages.AccountID) (messages.Author, error) {
	if f.err != nil {
		return messages.Author{}, f.err
	}
	return messages.AuthorOf("le professeur"), nil
}

func in() announce_rally_usecase.In {
	return announce_rally_usecase.In{
		Rally: "00000000-0000-7000-8000-000000000001", Account: professor, Country: "fr", Ground: "fr", Landmass: 7, At: at,
	}
}

func TestARallyIsToldInTheFlagsRoomUnderTheCallersName(t *testing.T) {
	store, updates := inmemory_announcement_storage.New(), &recordedFeed{}

	require.NoError(t, announce_rally_usecase.New(fakeAuthors{}, announce_usecase.New(store, updates)).Execute(t.Context(), in()))

	kept := store.Kept()
	require.Len(t, kept, 1)
	assert.Equal(t, announcements.KindRally, kept[0].Kind())
	assert.Equal(t, messages.FlagRoom("fr"), kept[0].Room())
	assert.Equal(t, at, kept[0].At())
	assert.JSONEq(t, `{"name":"le professeur","country":"fr","ground":"fr","landmass":7}`, string(kept[0].Payload()))
	require.Len(t, updates.updates, 1, "every open chat of the room sees it at once")
}

func TestARallyNobodyCanNameIsNotTold(t *testing.T) {
	store := inmemory_announcement_storage.New()

	err := announce_rally_usecase.New(fakeAuthors{err: errors.New("player down")}, announce_usecase.New(store, &recordedFeed{})).
		Execute(t.Context(), in())

	require.Error(t, err, "the bus tries it again")
	assert.Empty(t, store.Kept())
}

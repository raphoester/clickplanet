package mark_seen_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen/inmemory_seen_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen/usecases/mark_seen_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	ada = messages.AccountID{15: 1}
)

func markSeen(t *testing.T, store *inmemory_seen_storage.Storage, in mark_seen_usecase.In) error {
	t.Helper()

	return mark_seen_usecase.New(store, messages.NewRooms(messages.FakeFlags{ada: "fr"}), cptime.NewFixedClock(now)).
		Execute(t.Context(), in)
}

func TestTheMarkIsKeptForTheAccount(t *testing.T) {
	store := inmemory_seen_storage.New()

	err := markSeen(t, store, mark_seen_usecase.In{Account: ada, Room: messages.World, At: now.Add(-time.Minute)})

	require.NoError(t, err)
	assert.Equal(t, now.Add(-time.Minute), store.Kept(ada, messages.World))
}

func TestAMarkAheadOfTheServerIsKeptAsNow(t *testing.T) {
	store := inmemory_seen_storage.New()

	require.NoError(t, markSeen(t, store, mark_seen_usecase.In{Account: ada, Room: messages.World, At: now.Add(time.Hour)}))

	assert.Equal(t, now, store.Kept(ada, messages.World))
}

func TestNoAccountIsRefusedAndNothingIsKept(t *testing.T) {
	store := inmemory_seen_storage.New()

	err := markSeen(t, store, mark_seen_usecase.In{Account: messages.NoAccount, Room: messages.World, At: now})

	require.ErrorIs(t, err, messages.ErrNoAccount)
	assert.True(t, store.Kept(messages.NoAccount, messages.World).IsZero())
}

func TestNoTimeIsRefused(t *testing.T) {
	err := markSeen(t, inmemory_seen_storage.New(), mark_seen_usecase.In{Account: ada, Room: messages.World})

	assert.ErrorIs(t, err, seen.ErrNoTime)
}

func TestAStoreFailureIsAnError(t *testing.T) {
	store := inmemory_seen_storage.New()
	store.FailWith(errors.New("postgres is down"))

	err := markSeen(t, store, mark_seen_usecase.In{Account: ada, Room: messages.World, At: now})

	assert.ErrorContains(t, err, "postgres is down")
}

func TestEachRoomKeepsItsOwnMark(t *testing.T) {
	store := inmemory_seen_storage.New()

	require.NoError(t, markSeen(t, store, mark_seen_usecase.In{Account: ada, Room: messages.World, At: now}))
	require.NoError(t, markSeen(t, store,
		mark_seen_usecase.In{Account: ada, Room: messages.FlagRoom("fr"), At: now.Add(-time.Hour)}))

	assert.Equal(t, now, store.Kept(ada, messages.World))
	assert.Equal(t, now.Add(-time.Hour), store.Kept(ada, messages.FlagRoom("fr")))
}

func TestNoMarkIsKeptForTheRoomOfAnotherFlag(t *testing.T) {
	store := inmemory_seen_storage.New()

	err := markSeen(t, store, mark_seen_usecase.In{Account: ada, Room: messages.FlagRoom("de"), At: now})

	require.ErrorIs(t, err, messages.ErrNotInRoom)
	assert.True(t, store.Kept(ada, messages.FlagRoom("de")).IsZero())
}

package seen_get_history_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/get_history_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/get_history_usecase/seen_get_history"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen/inmemory_seen_storage"
)

var (
	at  = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	ada = messages.AccountID{15: 1}
)

type stubHistory struct {
	history get_history_usecase.History
	err     error
}

func (s stubHistory) Execute(context.Context, messages.AccountID) (get_history_usecase.History, error) {
	return s.history, s.err
}

var shown = get_history_usecase.History{Messages: []get_history_usecase.Entry{{Message: messages.Message{ID: "m-1"}}}}

func TestTheHistorySaysWhenTheCallerLastSawTheChat(t *testing.T) {
	store := inmemory_seen_storage.New()
	require.NoError(t, store.SaveSeen(t.Context(), ada, at))

	history, err := seen_get_history.New(stubHistory{history: shown}, store).Execute(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, at, history.SeenUntil)
	assert.Equal(t, shown.Messages, history.Messages)
}

func TestACallerWithNoMarkHasSeenNothing(t *testing.T) {
	history, err := seen_get_history.New(stubHistory{history: shown}, inmemory_seen_storage.New()).Execute(t.Context(), ada)

	require.NoError(t, err)
	assert.True(t, history.SeenUntil.IsZero())
}

func TestNoAccountIsNotAskedFor(t *testing.T) {
	store := inmemory_seen_storage.New()
	store.FailWith(errors.New("nobody should ask"))

	history, err := seen_get_history.New(stubHistory{history: shown}, store).Execute(t.Context(), messages.NoAccount)

	require.NoError(t, err)
	assert.True(t, history.SeenUntil.IsZero())
}

func TestAStoreFailureFailsTheHistory(t *testing.T) {
	store := inmemory_seen_storage.New()
	store.FailWith(errors.New("postgres is down"))

	_, err := seen_get_history.New(stubHistory{history: shown}, store).Execute(t.Context(), ada)

	assert.ErrorContains(t, err, "postgres is down")
}

func TestAHistoryThatFailedIsNotRead(t *testing.T) {
	_, err := seen_get_history.New(stubHistory{err: errors.New("no history")}, inmemory_seen_storage.New()).Execute(t.Context(), ada)

	assert.ErrorContains(t, err, "no history")
}

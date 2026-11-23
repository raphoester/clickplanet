package close_note_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations/inmemory_conversation_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations/usecases/close_note_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

var (
	note = notes.NoteID{15: 1}
	now  = time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
)

func TestANoteClosedKeepsItsConversationAndWhenItWasClosed(t *testing.T) {
	store := inmemory_conversation_storage.New()
	store.KeepNote(note)
	answered := conversations.Snapshot{Note: note, Answers: []conversations.Answer{
		{Text: "Thanks", Mail: "a1@clickplanet.lol", At: now.Add(-time.Hour)},
	}}
	store.Seed(answered)

	closedAt, err := close_note_usecase.New(store, cptime.NewFixedClock(now)).Execute(t.Context(), note)

	require.NoError(t, err)
	assert.Equal(t, now, closedAt)
	answered.ClosedAt = now
	assert.Equal(t, []conversations.Snapshot{answered}, store.Kept())
}

func TestANoteNoLongerKeptIsNotClosed(t *testing.T) {
	_, err := close_note_usecase.New(inmemory_conversation_storage.New(), cptime.NewFixedClock(now)).Execute(t.Context(), note)

	require.ErrorIs(t, err, notes.ErrUnknownNote)
}

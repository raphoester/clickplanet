package forget_notes_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/inmemory_note_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/forget_notes_usecase"
)

var (
	at  = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	ada = notes.AccountID{15: 1}
	bob = notes.AccountID{15: 2}
)

func suggestion(n byte, account notes.AccountID, text string) notes.Snapshot {
	return notes.Snapshot{ID: notes.NoteID{15: n}, Kind: notes.KindSuggestion, Account: account, Text: text, At: at}
}

func TestAForgottenAccountLosesEveryNoteItSentAndTheOthersStay(t *testing.T) {
	store := inmemory_note_storage.New()
	kept := suggestion(2, bob, "more flags")
	store.Seed(suggestion(1, ada, "a night mode"), kept, suggestion(3, ada, "a day mode"))

	require.NoError(t, forget_notes_usecase.New(store).Execute(t.Context(), ada))

	assert.Equal(t, []notes.Snapshot{kept}, store.Kept())
}

func TestAnAccountThatSentNothingIsForgottenWithNoError(t *testing.T) {
	store := inmemory_note_storage.New()
	kept := suggestion(1, bob, "more flags")
	store.Seed(kept)

	require.NoError(t, forget_notes_usecase.New(store).Execute(t.Context(), ada))

	assert.Equal(t, []notes.Snapshot{kept}, store.Kept())
}

func TestAFailureToForgetIsAnError(t *testing.T) {
	store := inmemory_note_storage.New()
	refused := errors.New("the database is down")
	store.FailWith(refused)

	assert.ErrorIs(t, forget_notes_usecase.New(store).Execute(t.Context(), ada), refused)
}

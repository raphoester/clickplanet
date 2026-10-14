package anonymize_notes_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/inmemory_note_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/anonymize_notes_usecase"
)

var (
	at  = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	ada = notes.AccountID{15: 1}
	bob = notes.AccountID{15: 2}
)

func suggestion(n byte, account notes.AccountID, text string) notes.Snapshot {
	return notes.Snapshot{ID: notes.NoteID{15: n}, Kind: notes.KindSuggestion, Account: account, Text: text, At: at}
}

func anonymous(n byte, text string) notes.Snapshot {
	return suggestion(n, notes.AccountID{}, text)
}

func TestEveryNoteAnAccountSentKeepsItsTextAndLosesItsSenderAndTheOthersAreLeftAlone(t *testing.T) {
	store := inmemory_note_storage.New()
	kept := suggestion(2, bob, "more flags")
	store.Seed(suggestion(1, ada, "a night mode"), kept, suggestion(3, ada, "a day mode"))

	require.NoError(t, anonymize_notes_usecase.New(store).Execute(t.Context(), ada))

	assert.ElementsMatch(t, []notes.Snapshot{anonymous(1, "a night mode"), kept, anonymous(3, "a day mode")}, store.Kept())
}

func TestAnAccountThatSentNothingChangesNothing(t *testing.T) {
	store := inmemory_note_storage.New()
	kept := suggestion(1, bob, "more flags")
	store.Seed(kept)

	require.NoError(t, anonymize_notes_usecase.New(store).Execute(t.Context(), ada))

	assert.Equal(t, []notes.Snapshot{kept}, store.Kept())
}

func TestAFailureToAnonymizeIsAnError(t *testing.T) {
	store := inmemory_note_storage.New()
	refused := errors.New("the database is down")
	store.FailWith(refused)

	assert.ErrorIs(t, anonymize_notes_usecase.New(store).Execute(t.Context(), ada), refused)
}

func TestANoteKeepsItsFilesWhenItLosesItsSender(t *testing.T) {
	store := inmemory_note_storage.New()
	sent := suggestion(1, ada, "a night mode")
	sent.Attachments = []notes.Attachment{{MediaType: notes.MediaPNG, Size: 400}, {MediaType: notes.MediaMP4, Size: 9000}}
	store.Seed(sent)

	require.NoError(t, anonymize_notes_usecase.New(store).Execute(t.Context(), ada))

	want := sent
	want.Account = notes.AccountID{}
	assert.Equal(t, []notes.Snapshot{want}, store.Kept())
}

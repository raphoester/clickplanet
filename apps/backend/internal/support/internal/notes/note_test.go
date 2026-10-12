package notes_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

var sentAt = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

func valid() notes.Snapshot {
	return notes.Snapshot{ID: notes.NoteID{15: 1}, Kind: notes.KindSuggestion, Account: ada, Text: "a night mode", At: sentAt}
}

func TestASuggestionIsANoteOfThatKindWithItsTextCleaned(t *testing.T) {
	note, err := notes.NewSuggestion(notes.NoteID{15: 1}, ada, "  a night mode\tplease  ", sentAt)

	require.NoError(t, err)
	assert.Equal(t, notes.Snapshot{
		ID: notes.NoteID{15: 1}, Kind: notes.KindSuggestion, Account: ada, Text: "a night mode please", At: sentAt,
	}, note.TakeSnapshot())
}

func TestASuggestionWithAnInvalidTextCannotBeBuilt(t *testing.T) {
	_, err := notes.NewSuggestion(notes.NoteID{15: 1}, ada, "   ", sentAt)

	require.ErrorIs(t, err, notes.ErrInvalidText)
}

func TestASnapshotRestoresTheNoteItWasTakenFrom(t *testing.T) {
	note, err := valid().Restore()
	require.NoError(t, err)

	again, err := note.TakeSnapshot().Restore()

	require.NoError(t, err)
	assert.Equal(t, note, again)
	assert.Equal(t, valid(), again.TakeSnapshot())
}

func TestASnapshotThatBreaksARuleRestoresNothing(t *testing.T) {
	for name, broken := range map[string]func(*notes.Snapshot){
		"no id":           func(s *notes.Snapshot) { s.ID = notes.NoteID{} },
		"an unknown kind": func(s *notes.Snapshot) { s.Kind = "praise" },
		"no kind":         func(s *notes.Snapshot) { s.Kind = "" },
		"no account":      func(s *notes.Snapshot) { s.Account = notes.AccountID{} },
		"no time":         func(s *notes.Snapshot) { s.At = time.Time{} },
	} {
		snapshot := valid()
		broken(&snapshot)

		_, err := snapshot.Restore()

		require.ErrorIs(t, err, notes.ErrInvalidNote, name)
	}
}

func TestASnapshotWithAnInvalidTextRestoresNothing(t *testing.T) {
	snapshot := valid()
	snapshot.Text = " \n "

	_, err := snapshot.Restore()

	require.ErrorIs(t, err, notes.ErrInvalidText)
	assert.NotErrorIs(t, err, notes.ErrInvalidNote)
}

func TestAForgottenNoteSaysSoInItsSnapshotAndKeepsTheRest(t *testing.T) {
	note, err := valid().Restore()
	require.NoError(t, err)

	forgotten := note.Forgotten()

	want := valid()
	want.Forgotten = true
	assert.Equal(t, want, forgotten.TakeSnapshot())
	assert.False(t, note.TakeSnapshot().Forgotten, "the note it was made from is left as it was")
}

func TestEveryNoteOfAListIsForgottenInItsOrder(t *testing.T) {
	first, err := valid().Restore()
	require.NoError(t, err)
	other := valid()
	other.ID = notes.NoteID{15: 2}
	second, err := other.Restore()
	require.NoError(t, err)

	forgotten := notes.AllForgotten([]notes.Note{first, second})

	require.Len(t, forgotten, 2)
	assert.Equal(t, first.Forgotten(), forgotten[0])
	assert.Equal(t, second.Forgotten(), forgotten[1])
	assert.Empty(t, notes.AllForgotten(nil))
}

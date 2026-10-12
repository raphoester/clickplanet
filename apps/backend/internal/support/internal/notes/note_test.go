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
	note, err := notes.NewSuggestion(notes.NoteID{15: 1}, ada, "  a night mode\tplease  ", notes.Diagnostics{}, sentAt)

	require.NoError(t, err)
	assert.Equal(t, notes.Snapshot{
		ID: notes.NoteID{15: 1}, Kind: notes.KindSuggestion, Account: ada, Text: "a night mode please", At: sentAt,
	}, note.TakeSnapshot())
}

func TestASuggestionNeedsTheAccountThatSendsIt(t *testing.T) {
	_, err := notes.NewSuggestion(notes.NoteID{15: 1}, notes.AccountID{}, "a night mode", notes.Diagnostics{}, sentAt)

	require.ErrorIs(t, err, notes.ErrInvalidNote)
}

func TestASuggestionWithAnInvalidTextCannotBeBuilt(t *testing.T) {
	_, err := notes.NewSuggestion(notes.NoteID{15: 1}, ada, "   ", notes.Diagnostics{}, sentAt)

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

func TestAnAnonymizedNoteKeepsItsTextAndNothingSaysWhoSentIt(t *testing.T) {
	note, err := valid().Restore()
	require.NoError(t, err)

	anonymized := note.Anonymized()

	want := valid()
	want.Account = notes.AccountID{}
	assert.Equal(t, want, anonymized.TakeSnapshot())
	assert.Equal(t, valid(), note.TakeSnapshot(), "the note it was made from is left as it was")
	assert.Equal(t, anonymized, anonymized.Anonymized(), "twice is once")
}

func TestAnAnonymizedNoteIsRestoredAsItWasKept(t *testing.T) {
	kept := valid()
	kept.Account = notes.AccountID{}

	note, err := kept.Restore()

	require.NoError(t, err)
	assert.Equal(t, kept, note.TakeSnapshot())
}

func TestEveryNoteOfAListIsAnonymizedInItsOrder(t *testing.T) {
	first, err := valid().Restore()
	require.NoError(t, err)
	other := valid()
	other.ID = notes.NoteID{15: 2}
	second, err := other.Restore()
	require.NoError(t, err)

	anonymized := notes.AllAnonymized([]notes.Note{first, second})

	require.Len(t, anonymized, 2)
	assert.Equal(t, first.Anonymized(), anonymized[0])
	assert.Equal(t, second.Anonymized(), anonymized[1])
	assert.Empty(t, notes.AllAnonymized(nil))
}

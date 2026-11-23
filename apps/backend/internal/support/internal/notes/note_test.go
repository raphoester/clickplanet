package notes_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

var (
	ada    = notes.AccountID{15: 1}
	sentAt = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
)

func valid() notes.Snapshot {
	return notes.Snapshot{ID: notes.NoteID{15: 1}, Kind: notes.KindSuggestion, Account: ada, Text: "a night mode", At: sentAt}
}

func TestASuggestionNeedsTheAccountThatSendsIt(t *testing.T) {
	_, err := notes.NewSuggestion(notes.NoteID{15: 1}, notes.Sent{Text: "a night mode", ReplyAddress: "ada@example.com"}, sentAt)

	require.ErrorIs(t, err, notes.ErrInvalidNote)
}

func TestANoteKeepsTheAddressItIsAnsweredAtInLowerCase(t *testing.T) {
	note, err := notes.NewBugReport(notes.NoteID{15: 1},
		notes.Sent{Account: ada, Text: "the globe is black", ReplyAddress: "Ada@Example.com"}, sentAt)
	require.NoError(t, err)

	assert.Equal(t, notes.Snapshot{
		ID: notes.NoteID{15: 1}, Kind: notes.KindBug, Account: ada, Text: "the globe is black", ReplyAddress: "ada@example.com",
		At: sentAt,
	}, note.TakeSnapshot())
}

func TestAnAnonymizedNoteHasNoSenderAndNoAddress(t *testing.T) {
	snapshot := valid()
	snapshot.ReplyAddress = "ada@example.com"
	note, err := snapshot.Restore()
	require.NoError(t, err)

	anonymized := note.Anonymized().TakeSnapshot()

	snapshot.Account, snapshot.ReplyAddress = notes.AccountID{}, ""
	assert.Equal(t, snapshot, anonymized)
}

func TestASnapshotThatBreaksARuleRestoresNothing(t *testing.T) {
	for name, broken := range map[string]func(*notes.Snapshot){
		"no id":           func(s *notes.Snapshot) { s.ID = notes.NoteID{} },
		"an unknown kind": func(s *notes.Snapshot) { s.Kind = "praise" },
		"no kind":         func(s *notes.Snapshot) { s.Kind = "" },
		"no time":         func(s *notes.Snapshot) { s.At = time.Time{} },
		"a bad address":   func(s *notes.Snapshot) { s.ReplyAddress = "ada@localhost" },
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

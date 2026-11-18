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
	_, err := notes.NewSuggestion(notes.NoteID{15: 1}, notes.AccountID{}, "a night mode", notes.Diagnostics{}, sentAt)

	require.ErrorIs(t, err, notes.ErrInvalidNote)
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

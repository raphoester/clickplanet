package notes_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

func diagnosticsKept(t *testing.T, said notes.Diagnostics) notes.Diagnostics {
	t.Helper()

	note, err := notes.NewBugReport(notes.NoteID{15: 1}, ada, "the globe is black", said, sentAt)
	require.NoError(t, err)
	return note.TakeSnapshot().Diagnostics
}

func TestABugReportKeepsWhatThePageSaid(t *testing.T) {
	phone := notes.Diagnostics{
		Build: "play-a1b2c3.js", ViewportWidth: 390, ViewportHeight: 844, ConnectionDown: true, UserAgent: "Mozilla/5.0 (iPhone)",
	}

	assert.Equal(t, phone, diagnosticsKept(t, phone))
}

func TestABugReportKeepsItsDiagnosticsCutAndCleaned(t *testing.T) {
	kept := diagnosticsKept(t, notes.Diagnostics{
		Build:          " play\x00-\n" + strings.Repeat("a", 100),
		ViewportWidth:  4_000_000_000,
		ViewportHeight: 4_000_000_000,
		UserAgent:      strings.Repeat("🌍", 300) + string([]byte{0xff}),
	})

	assert.Equal(t, "play-"+strings.Repeat("a", 59), kept.Build, "64 characters, with no control character")
	assert.Equal(t, uint32(100_000), kept.ViewportWidth)
	assert.Equal(t, uint32(100_000), kept.ViewportHeight)
	assert.Equal(t, 256, utf8.RuneCountInString(kept.UserAgent))
	assert.True(t, utf8.ValidString(kept.UserAgent))
}

func TestABugReportWithNothingSaidKeepsNoDiagnostics(t *testing.T) {
	assert.Equal(t, notes.Diagnostics{}, diagnosticsKept(t, notes.Diagnostics{Build: " \n", UserAgent: "\x00"}))
}

func TestABugReportIsANoteOfThatKindWithItsTextCleaned(t *testing.T) {
	note, err := notes.NewBugReport(notes.NoteID{15: 1}, ada, "  the globe is black  ", notes.Diagnostics{}, sentAt)

	require.NoError(t, err)
	assert.Equal(t, notes.Snapshot{
		ID: notes.NoteID{15: 1}, Kind: notes.KindBug, Account: ada, Text: "the globe is black", At: sentAt,
	}, note.TakeSnapshot())
}

func TestOnlyABugReportHasDiagnostics(t *testing.T) {
	snapshot := valid()
	snapshot.Diagnostics = notes.Diagnostics{Build: "play-a1b2c3.js"}

	_, err := snapshot.Restore()

	require.ErrorIs(t, err, notes.ErrInvalidNote)
}

func TestABugReportNeedsTheAccountThatSendsIt(t *testing.T) {
	_, err := notes.NewBugReport(notes.NoteID{15: 1}, notes.AccountID{}, "the globe is black", notes.Diagnostics{}, sentAt)

	require.ErrorIs(t, err, notes.ErrInvalidNote)
}

func TestAnAnonymizedBugReportKeepsItsDiagnostics(t *testing.T) {
	said := notes.Diagnostics{Build: "play-a1b2c3.js", ViewportWidth: 390, ViewportHeight: 844, UserAgent: "Mozilla/5.0 (iPhone)"}
	note, err := notes.NewBugReport(notes.NoteID{15: 1}, ada, "the globe is black", said, sentAt)
	require.NoError(t, err)

	anonymized := note.Anonymized().TakeSnapshot()

	assert.Equal(t, notes.AccountID{}, anonymized.Account)
	assert.Equal(t, said, anonymized.Diagnostics)
}

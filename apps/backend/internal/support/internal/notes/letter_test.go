package notes_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

func note(t *testing.T, kind notes.Kind, address cpmail.Address) notes.Note {
	t.Helper()

	n, err := notes.Snapshot{
		ID: notes.NoteID{15: 1}, Kind: kind, Account: ada, Text: "the globe is black\non my phone", ReplyAddress: address, At: sentAt,
	}.Restore()
	require.NoError(t, err)
	return n
}

func TestAnAnswerGoesToTheNotesAddressAndQuotesTheNote(t *testing.T) {
	thread := []cpmail.MessageID{"a1@clickplanet.lol", "r1@example.com"}

	letter, err := note(t, notes.KindBug, "ada@example.com").Answer("  Fixed, reload the page.  ", thread)

	require.NoError(t, err)
	assert.Equal(t, notes.Letter{
		Note: notes.NoteID{15: 1}, To: "ada@example.com", Subject: "Re: your bug report",
		Text:   "Fixed, reload the page.\n\nOn 11 October 2026, you wrote:\n> the globe is black\n> on my phone\n",
		Thread: thread,
	}, letter)
}

func TestTheSubjectSaysTheKindOfTheNote(t *testing.T) {
	letter, err := note(t, notes.KindSuggestion, "ada@example.com").Answer("Thanks!", nil)

	require.NoError(t, err)
	assert.Equal(t, "Re: your idea", letter.Subject)
}

func TestANoteWithNoAddressOrAnAnswerWithNoWordsGetsNoLetter(t *testing.T) {
	_, err := note(t, notes.KindBug, "").Answer("Fixed", nil)
	require.ErrorIs(t, err, notes.ErrNoReplyAddress)

	_, err = note(t, notes.KindBug, "ada@example.com").Answer("  ", nil)
	require.ErrorIs(t, err, notes.ErrInvalidText)
}

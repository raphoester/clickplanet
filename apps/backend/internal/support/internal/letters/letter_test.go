package letters_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

var (
	at    = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	ada   = notes.AccountID{15: 1}
	id    = letters.LetterID{15: 1}
	about = letters.SubmissionID{15: 1}
	inbox = notes.NewInbox("support@clickplanet.lol")
)

func TestALettersAddressIsUnderTheInboxAndNeverANotes(t *testing.T) {
	address := letters.AddressOf(inbox, id)
	assert.Equal(t, cpmail.Address("support+l-00000000-0000-0000-0000-000000000001@clickplanet.lol"), address)

	found, err := letters.LetterOf(inbox, string(address))
	require.NoError(t, err)
	assert.Equal(t, id, found)
	_, err = inbox.NoteOf(string(address))
	require.ErrorIs(t, err, notes.ErrNotANoteAddress, "a reply to a letter is never a note's")

	note := inbox.Of(notes.NoteID{15: 1})
	_, err = letters.LetterOf(inbox, string(note))
	require.ErrorIs(t, err, letters.ErrNotALetter, "a note's address names no letter")
	_, err = letters.LetterOf(inbox, "support@clickplanet.lol")
	require.ErrorIs(t, err, letters.ErrNotALetter)
}

func TestTheFirstMailGoesUnderItsSubjectAndTheNextInReplyToIt(t *testing.T) {
	draft, err := letters.NewDraft(id, about, ada, "ada@example.com", "  Your flag Catalonia ", nil)
	require.NoError(t, err)

	first, err := draft.Mail("Which city is its capital?")
	require.NoError(t, err)
	assert.Equal(t, letters.Mail{
		Letter: id, To: "ada@example.com", Subject: "Your flag Catalonia", Text: "Which city is its capital?",
		Thread: []cpmail.MessageID{},
	}, first)

	letter, err := draft.Sent(first.Text, "l1@clickplanet.lol", at)
	require.NoError(t, err)
	replied, err := letter.Replied(conversations.Reply{
		Mail: "r1@mail.example.com", From: "ada@example.com", Text: "Barcelona.", At: at.Add(time.Hour),
	})
	require.NoError(t, err)
	next, err := replied.Mail("Thanks!")
	require.NoError(t, err)
	assert.Equal(t, "Re: Your flag Catalonia", next.Subject)
	assert.Equal(t, []cpmail.MessageID{"l1@clickplanet.lol", "r1@mail.example.com"}, next.Thread)
}

func TestALettersSubjectAndTextFollowTheirRules(t *testing.T) {
	for _, subject := range []string{"", "  ", strings.Repeat("a", 201), "a\nb"} {
		_, err := letters.NewDraft(id, about, ada, "ada@example.com", subject, nil)
		require.ErrorIs(t, err, letters.ErrInvalidSubject, "%q", subject)
	}
	_, err := letters.NewDraft(id, about, ada, "", "Your flag", nil)
	require.ErrorIs(t, err, notes.ErrNoReplyAddress, "an account with no address gets no letter")
	_, err = letters.NewDraft(id, letters.SubmissionID{}, ada, "ada@example.com", "Your flag", nil)
	require.ErrorIs(t, err, letters.ErrInvalidSubmission, "a letter is about a submission")

	draft, err := letters.NewDraft(id, about, ada, "ada@example.com", "Your flag", nil)
	require.NoError(t, err)
	_, err = draft.Mail(strings.Repeat("a", 5001))
	require.ErrorIs(t, err, notes.ErrInvalidText)
}

func TestASubmissionHasOneLetterAtMost(t *testing.T) {
	catalonia := letters.Restored(letters.TestWritten(id, ada, at))
	other := letters.Restored(letters.TestWritten(letters.LetterID{15: 2}, ada, at))

	_, err := letters.NewDraft(letters.LetterID{15: 3}, catalonia.TakeSnapshot().Submission, ada, "ada@example.com",
		"Your flag Catalonia", []letters.Letter{other, catalonia})
	require.ErrorIs(t, err, letters.ErrWrittenAlready)

	_, err = letters.NewDraft(letters.LetterID{15: 3}, letters.SubmissionID{15: 3}, ada, "ada@example.com",
		"Your flag Tyrol", []letters.Letter{other, catalonia})
	require.NoError(t, err, "another submission of the same founder has its own")
}

func TestAReplyIsKeptOnce(t *testing.T) {
	letter := letters.Restored(letters.TestWritten(id, ada, at))
	reply := conversations.Reply{Mail: "r1@mail.example.com", From: "ada@example.com", Text: "Barcelona.", At: at}

	replied, err := letter.Replied(reply)
	require.NoError(t, err)
	_, err = replied.Replied(reply)
	require.ErrorIs(t, err, conversations.ErrReceivedAlready)
}

func TestAnAnonymizedLetterTakesNoMailAndNamesNobody(t *testing.T) {
	letter := letters.Restored(letters.TestWritten(id, ada, at))
	replied, err := letter.Replied(conversations.Reply{
		Mail: "r1@mail.example.com", From: "ada@example.com", Text: "Barcelona.", At: at,
	})
	require.NoError(t, err)

	anonymized := replied.Anonymized().TakeSnapshot()
	assert.Equal(t, notes.AccountID{}, anonymized.Account)
	assert.Empty(t, anonymized.To)
	assert.Empty(t, anonymized.Replies[0].From)
	assert.Equal(t, "Barcelona.", anonymized.Replies[0].Text, "what was said stays")
	_, err = replied.Anonymized().Mail("Hello?")
	require.ErrorIs(t, err, notes.ErrNoReplyAddress)
}

func TestALetterKeptWithNoMailOrHalfAnonymizedIsRefused(t *testing.T) {
	for name, edit := range map[string]func(*letters.Snapshot){
		"no mail went":           func(s *letters.Snapshot) { s.Sent = nil },
		"an account, no address": func(s *letters.Snapshot) { s.To = "" },
		"an address, no account": func(s *letters.Snapshot) { s.Account = notes.AccountID{} },
		"no id":                  func(s *letters.Snapshot) { s.ID = letters.LetterID{} },
		"no submission":          func(s *letters.Snapshot) { s.Submission = letters.SubmissionID{} },
		"no subject":             func(s *letters.Snapshot) { s.Subject = "" },
	} {
		snapshot := letters.TestWritten(id, ada, at)
		edit(&snapshot)
		_, err := snapshot.Restore()
		require.Error(t, err, name)
	}
}

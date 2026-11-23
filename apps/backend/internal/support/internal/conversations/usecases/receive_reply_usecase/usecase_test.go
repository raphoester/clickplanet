package receive_reply_usecase_test

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations/inmemory_conversation_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations/usecases/receive_reply_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/inmemory_note_storage"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

var (
	now   = time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	inbox = notes.NewInbox("support@clickplanet.lol")
	note  = notes.Snapshot{
		ID: notes.NoteID{15: 1}, Kind: notes.KindBug, Account: notes.AccountID{15: 1}, Text: "the globe is black",
		ReplyAddress: "ada@example.com", At: now.Add(-24 * time.Hour),
	}
	reply = conversations.Received{
		Mail: "r1@mail.example.com", From: "ada@example.com",
		Text: "An iPhone 13.\n\nOn Mon, ClickPlanet <support+00000000-0000-0000-0000-000000000001@clickplanet.lol> wrote:\n> Which phone?",
	}
)

type stubReader struct {
	received conversations.Received
	err      error
}

func (s stubReader) Read(io.Reader) (conversations.Received, error) {
	return s.received, s.err
}

type testSuite struct {
	suite.Suite

	notes         *inmemory_note_storage.Storage
	conversations *inmemory_conversation_storage.Storage
}

func (s *testSuite) SetupTest() {
	s.notes = inmemory_note_storage.New()
	s.notes.Seed(note)
	s.conversations = inmemory_conversation_storage.New()
	s.conversations.KeepNote(note.ID)
}

func (s *testSuite) receive(reader stubReader, recipient string) error {
	_, err := receive_reply_usecase.New(inbox, reader, s.notes, s.conversations, cptime.NewFixedClock(now)).
		Execute(s.T().Context(), receive_reply_usecase.In{Recipient: recipient, Raw: strings.NewReader("")})
	return err //nolint:wrapcheck // the test reads the sentinel.
}

func (s *testSuite) TestAReplyToANoteIsKeptWithoutTheMailItQuotes() {
	s.Require().NoError(s.receive(stubReader{received: reply}, string(inbox.Of(note.ID))))

	s.Equal([]conversations.Snapshot{{Note: note.ID, Replies: []conversations.Reply{
		{Mail: "r1@mail.example.com", From: "ada@example.com", Text: "An iPhone 13.", At: now},
	}}}, s.conversations.Kept())
}

func (s *testSuite) TestTheSameMailTwiceIsKeptOnce() {
	s.Require().NoError(s.receive(stubReader{received: reply}, string(inbox.Of(note.ID))))

	s.Require().ErrorIs(s.receive(stubReader{received: reply}, string(inbox.Of(note.ID))), conversations.ErrReceivedAlready)
	s.Len(s.conversations.Kept()[0].Replies, 1)
}

func (s *testSuite) TestAMailToNoNoteOrToANoteThatTakesNoneIsDropped() {
	anonymized := note
	anonymized.ID, anonymized.Account, anonymized.ReplyAddress = notes.NoteID{15: 2}, notes.AccountID{}, ""
	s.notes.Seed(anonymized)

	for recipient, want := range map[string]error{
		"support@clickplanet.lol":             notes.ErrNotANoteAddress,
		string(inbox.Of(notes.NoteID{15: 9})): notes.ErrUnknownNote,
		string(inbox.Of(anonymized.ID)):       notes.ErrNoReplyAddress,
	} {
		err := s.receive(stubReader{received: reply}, recipient)

		s.Require().ErrorIs(err, want, recipient)
		s.True(receive_reply_usecase.IsDropped(err), recipient)
	}
	s.Empty(s.conversations.Kept())
}

func (s *testSuite) TestAMailThatCannotBeReadIsDropped() {
	err := s.receive(stubReader{err: conversations.ErrUnreadable}, string(inbox.Of(note.ID)))

	s.Require().ErrorIs(err, conversations.ErrUnreadable)
	s.True(receive_reply_usecase.IsDropped(err))
}

func (s *testSuite) TestAReplyThatCouldNotBeKeptIsNotDropped() {
	s.conversations.FailWith(io.ErrUnexpectedEOF)

	err := s.receive(stubReader{received: reply}, string(inbox.Of(note.ID)))

	s.Require().ErrorIs(err, io.ErrUnexpectedEOF)
	s.False(receive_reply_usecase.IsDropped(err))
}

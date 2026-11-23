package answer_note_usecase_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations/inmemory_conversation_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations/usecases/answer_note_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/inmemory_note_storage"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

var (
	sentAt = time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)
	now    = time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	note   = notes.Snapshot{
		ID: notes.NoteID{15: 1}, Kind: notes.KindBug, Account: notes.AccountID{15: 1}, Text: "the globe is black",
		ReplyAddress: "ada@example.com", At: sentAt,
	}
)

type stubPost struct {
	sent []notes.Letter
	err  error
}

func (s *stubPost) Send(_ context.Context, letter notes.Letter) (cpmail.MessageID, error) {
	if s.err != nil {
		return "", s.err
	}
	s.sent = append(s.sent, letter)
	return cpmail.MessageID(fmt.Sprintf("a%d@clickplanet.lol", len(s.sent))), nil
}

type testSuite struct {
	suite.Suite

	notes         *inmemory_note_storage.Storage
	conversations *inmemory_conversation_storage.Storage
	post          *stubPost
	useCase       *answer_note_usecase.UseCase
}

func (s *testSuite) SetupTest() {
	s.notes = inmemory_note_storage.New()
	s.notes.Seed(note)
	s.conversations = inmemory_conversation_storage.New()
	s.conversations.KeepNote(note.ID)
	s.post = &stubPost{}
	s.useCase = answer_note_usecase.New(s.notes, s.conversations, s.post, cptime.NewFixedClock(now))
}

func (s *testSuite) answer(text string) (time.Time, error) {
	return s.useCase.Execute(s.T().Context(), answer_note_usecase.In{Note: note.ID, Text: text}) //nolint:wrapcheck // the test reads the sentinel.
}

func (s *testSuite) TestTheAnswerIsMailedThenKeptWithItsMail() {
	at, err := s.answer("Fixed, reload the page.")
	s.Require().NoError(err)

	s.Equal(now, at)
	s.Require().Len(s.post.sent, 1)
	s.Equal(cpmail.Address("ada@example.com"), s.post.sent[0].To)
	s.Empty(s.post.sent[0].Thread)
	s.Equal([]conversations.Snapshot{{Note: note.ID, Answers: []conversations.Answer{
		{Text: "Fixed, reload the page.", Mail: "a1@clickplanet.lol", At: now},
	}}}, s.conversations.Kept())
}

func (s *testSuite) TestALaterAnswerIsMailedInTheThreadOfTheEarlierMails() {
	s.conversations.Seed(conversations.Snapshot{
		Note:    note.ID,
		Answers: []conversations.Answer{{Text: "Which phone?", Mail: "a0@clickplanet.lol", At: sentAt.Add(time.Hour)}},
		Replies: []conversations.Reply{{Mail: "r1@example.com", From: "ada@example.com", Text: "An iPhone", At: sentAt.Add(2 * time.Hour)}},
	})

	_, err := s.answer("Fixed.")
	s.Require().NoError(err)

	s.Equal([]cpmail.MessageID{"a0@clickplanet.lol", "r1@example.com"}, s.post.sent[0].Thread)
	s.Len(s.conversations.Kept()[0].Answers, 2)
}

func (s *testSuite) TestAMailThatDidNotGoKeepsNothing() {
	s.post.err = fmt.Errorf("%w: cloudflare answered 500", notes.ErrNotSent)

	_, err := s.answer("Fixed.")

	s.Require().ErrorIs(err, notes.ErrNotSent)
	s.Empty(s.conversations.Kept())
}

func (s *testSuite) TestANoteWithNoAddressOrNoneAtAllIsNotAnswered() {
	anonymous := note
	anonymous.ID, anonymous.Account, anonymous.ReplyAddress = notes.NoteID{15: 2}, notes.AccountID{}, ""
	s.notes.Seed(anonymous)

	_, err := s.useCase.Execute(s.T().Context(), answer_note_usecase.In{Note: anonymous.ID, Text: "Fixed."})
	s.Require().ErrorIs(err, notes.ErrNoReplyAddress)

	_, err = s.useCase.Execute(s.T().Context(), answer_note_usecase.In{Note: notes.NoteID{15: 9}, Text: "Fixed."})
	s.Require().ErrorIs(err, notes.ErrUnknownNote)

	s.Empty(s.post.sent)
}

func (s *testSuite) TestAnAnswerWithNoWordsIsNotMailed() {
	_, err := s.answer("   ")

	s.Require().ErrorIs(err, notes.ErrInvalidText)
	s.Empty(s.post.sent)
}

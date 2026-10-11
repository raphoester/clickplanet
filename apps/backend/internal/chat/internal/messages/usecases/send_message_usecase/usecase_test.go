package send_message_usecase_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	appender  *fakeAppender
	publisher *fakePublisher
	authors   *fakeAuthors
	thread    *fakeThread
	clock     *cptime.FixedClock
	useCase   *send_message_usecase.UseCase
}

var (
	ada   = messages.AccountID{15: 1}
	guest = messages.AccountID{15: 2}
)

const firstID = messages.MessageID("00000000-0000-0000-0000-000000000001")

func (s *testSuite) SetupTest() {
	s.appender = &fakeAppender{}
	s.publisher = &fakePublisher{}
	s.authors = &fakeAuthors{names: map[messages.AccountID]string{ada: "Ada_L", guest: "guest_0b1c2d"}}
	s.thread = &fakeThread{shown: map[messages.MessageID]messages.Message{
		"parent": messages.Restored(s.T(), messages.Snapshot{
			ID: "parent", Account: ada, Author: messages.AuthorOf("Ada_L", false, 0, 0, messages.Title{}), Text: "who holds the north",
		}),
	}}
	s.clock = cptime.NewFixedClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	s.useCase = send_message_usecase.New(
		s.appender,
		s.publisher,
		s.authors,
		messages.NewDrafts(&messages.SequentialIDs{}, s.clock, messages.NewLimits(0), cpcountries.New()),
		s.thread,
	)
}

func (s *testSuite) send(in send_message_usecase.In) (messages.Snapshot, error) {
	message, err := s.useCase.Execute(cpctx.AddIPToContext(context.Background(), "1.2.3.4"), in)
	return message.TakeSnapshot(), err
}

func (s *testSuite) kept() []messages.RecordSnapshot {
	s.appender.mu.Lock()
	defer s.appender.mu.Unlock()

	kept := make([]messages.RecordSnapshot, 0, len(s.appender.records))
	for _, record := range s.appender.records {
		kept = append(kept, record.TakeSnapshot())
	}
	return kept
}

func (s *testSuite) published() []messages.Snapshot {
	published := make([]messages.Snapshot, 0, len(s.publisher.updates))
	for _, update := range s.publisher.updates {
		message, sent := update.Message()
		s.Require().True(sent)
		published = append(published, message.TakeSnapshot())
	}
	return published
}

func validIn() send_message_usecase.In {
	return send_message_usecase.In{
		Account:   guest,
		AuthorID:  "8f14e45f-ea23-4a1b-9c11-0b0d1a2b3c4d",
		CountryID: "fr",
		Text:      "hello planet",
		UserAgent: "curl/8",
	}
}

func (s *testSuite) TestNominalCase() {
	sent, err := s.send(validIn())
	s.Require().NoError(err)

	draft := messages.Snapshot{ID: firstID, SentAt: s.clock.Now(), Account: guest, Country: "fr", Text: "hello planet"}
	named := draft
	named.Author = messages.AuthorOf("guest_0b1c2d", false, 0, 0, messages.Title{})
	s.Equal(named, sent, "a guest posts under the name the player module gives it")
	s.Equal([]messages.AccountID{guest}, s.authors.asked)

	s.Equal([]messages.RecordSnapshot{{
		Message:   draft,
		AuthorID:  "8f14e45f-ea23-4a1b-9c11-0b0d1a2b3c4d",
		IP:        "1.2.3.4",
		UserAgent: "curl/8",
	}}, s.kept(), "what is kept is who sent it, and not a copy of what that account is called")

	s.Equal([]messages.Snapshot{named}, s.published(), "the message goes out once it is kept")
}

func (s *testSuite) TestEachMessageGetsAnIDOfItsOwn() {
	first, err := s.send(validIn())
	s.Require().NoError(err)
	second, err := s.send(validIn())
	s.Require().NoError(err)

	s.Equal(firstID, first.ID)
	s.Equal(messages.MessageID("00000000-0000-0000-0000-000000000002"), second.ID)
	kept := s.kept()
	s.Require().Len(kept, 2)
	s.Equal(first.ID, kept[0].Message.ID)
	s.Equal(second.ID, kept[1].Message.ID)
}

func (s *testSuite) TestTheSenderIPIsRecordedButNeverReturned() {
	sent, err := s.send(validIn())
	s.Require().NoError(err)

	s.Require().Len(s.kept(), 1)
	s.Equal("1.2.3.4", s.kept()[0].IP)

	s.NotContains(sent.Author.Name(), "1.2.3.4")
	s.NotContains(sent.Text, "1.2.3.4")
}

func (s *testSuite) TestTheMessageIsCleanedBeforeItIsStored() {
	in := validIn()
	in.Text = "hello\tplanet"

	sent, err := s.send(in)
	s.Require().NoError(err)
	s.Equal("hello planet", sent.Text)
	s.Equal("hello planet", s.kept()[0].Message.Text)
}

func (s *testSuite) TestAnInvalidTextIsRefusedAndNotStored() {
	in := validIn()
	in.Text = "   "

	_, err := s.send(in)
	s.Require().ErrorIs(err, messages.ErrInvalidMessage)
	s.Empty(s.kept())
}

func (s *testSuite) TestInvalidCountryIsRefused() {
	in := validIn()
	in.CountryID = "atlantis"

	_, err := s.send(in)
	s.Require().ErrorIs(err, messages.ErrInvalidMessage)
	s.Empty(s.kept())
}

func (s *testSuite) TestAPlainMessageAnswersNothing() {
	sent, err := s.send(validIn())
	s.Require().NoError(err)

	s.Equal(messages.Quote{}, sent.Quote)
	s.Equal(messages.Quote{}, s.kept()[0].Message.Quote)
}

func (s *testSuite) TestAReplyIsKeptAndGoesOutWithWhatItQuotes() {
	in := validIn()
	in.ReplyTo = "parent"

	sent, err := s.send(in)
	s.Require().NoError(err)

	quote := messages.Quote{
		MessageID: "parent", Author: messages.AuthorOf("Ada_L", false, 0, 0, messages.Title{}), Text: "who holds the north",
	}
	s.Equal(quote, sent.Quote)
	s.Require().Len(s.kept(), 1)
	s.Equal(quote, s.kept()[0].Message.Quote)
	s.Equal([]messages.Snapshot{sent}, s.published(), "everyone watching is shown what it answers with no second read")
}

func (s *testSuite) TestAReplyToAMessageThatIsNotShownIsRefusedAndNotStored() {
	in := validIn()
	in.ReplyTo = "gone"

	_, err := s.send(in)

	s.Require().ErrorIs(err, messages.ErrNotShown)
	s.Empty(s.authors.asked)
	s.Empty(s.kept())
	s.Empty(s.publisher.updates)
}

func (s *testSuite) TestAnInvalidReplyIsRefusedBeforeWhatItAnswersIsRead() {
	in := validIn()
	in.ReplyTo = "parent"
	in.Text = "   "

	_, err := s.send(in)

	s.Require().ErrorIs(err, messages.ErrInvalidMessage)
	s.Empty(s.thread.asked)
}

func (s *testSuite) TestStorageFailureFailsTheSend() {
	s.appender.err = errors.New("disk on fire")

	_, err := s.send(validIn())
	s.Require().Error(err)
	s.NotErrorIs(err, messages.ErrInvalidMessage)
	s.Empty(s.publisher.updates, "a message that cannot be kept is not sent")
}

func (s *testSuite) TestAPlayerWithAUsernamePostsUnderIt() {
	in := validIn()
	in.Account = ada

	sent, err := s.send(in)

	s.Require().NoError(err)
	s.Equal("Ada_L", sent.Author.Name())
	s.Equal(ada, s.kept()[0].Message.Account)
	s.Empty(s.kept()[0].Message.Author.Name(), "the name is read back from the account, not kept")
	s.Equal([]messages.AccountID{ada}, s.authors.asked)
}

func (s *testSuite) TestAnAdminPostsAsOneAndIsStoredAsOne() {
	s.authors.admins = map[messages.AccountID]bool{ada: true}
	in := validIn()
	in.Account = ada

	sent, err := s.send(in)

	s.Require().NoError(err)
	s.True(sent.Author.Admin(), "everyone watching is shown the crown at once")
	s.False(s.kept()[0].Message.Author.Admin(),
		"a player that stops being an admin stops looking like one on what it already said")
}

func (s *testSuite) TestAnAuthorThatCannotBeReadRefusesThePost() {
	s.authors.err = errors.New("the player module is stuck")

	_, err := s.send(validIn())

	s.Require().ErrorIs(err, messages.ErrAuthorUnavailable)
	s.NotErrorIs(err, messages.ErrInvalidMessage)
	s.Empty(s.kept())
}

func (s *testSuite) TestNoAccountIsRefusedBeforeAnythingIsAsked() {
	in := validIn()
	in.Account = cpsession.NoAccount

	_, err := s.send(in)

	s.Require().ErrorIs(err, messages.ErrNoAccount)
	s.Empty(s.authors.asked)
	s.Empty(s.kept())
}

type fakeAuthors struct {
	mu     sync.Mutex
	names  map[messages.AccountID]string
	admins map[messages.AccountID]bool
	err    error
	asked  []messages.AccountID
}

func (f *fakeAuthors) Author(_ context.Context, account messages.AccountID) (messages.Author, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.asked = append(f.asked, account)
	if f.err != nil {
		return messages.Author{}, f.err
	}
	return messages.AuthorOf(f.names[account], f.admins[account], 0, 0, messages.Title{}), nil
}

type fakeThread struct {
	shown map[messages.MessageID]messages.Message
	asked []messages.MessageID
}

func (f *fakeThread) Reply(_ context.Context, draft messages.Message, replyTo messages.MessageID) (messages.Message, error) {
	f.asked = append(f.asked, replyTo)
	if replyTo == "" {
		return draft, nil
	}
	answered, shown := f.shown[replyTo]
	if !shown {
		return messages.Message{}, messages.ErrNotShown
	}
	return draft.Answering(answered), nil
}

type fakePublisher struct {
	updates []feed.Update
}

func (f *fakePublisher) Publish(update feed.Update) {
	f.updates = append(f.updates, update)
}

type fakeAppender struct {
	mu      sync.Mutex
	records []messages.Record
	err     error
}

func (f *fakeAppender) Append(_ context.Context, record messages.Record) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.err != nil {
		return f.err
	}

	f.records = append(f.records, record)
	return nil
}

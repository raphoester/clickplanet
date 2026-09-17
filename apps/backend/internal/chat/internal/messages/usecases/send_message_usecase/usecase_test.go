package send_message_usecase_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	appender *fakeAppender
	authors  *fakeAuthors
	clock    *cptime.FixedClock
	useCase  *send_message_usecase.UseCase
}

var (
	ada   = messages.AccountID{15: 1}
	guest = messages.AccountID{15: 2}
)

func (s *testSuite) SetupTest() {
	s.appender = &fakeAppender{}
	s.authors = &fakeAuthors{names: map[messages.AccountID]string{ada: "Ada_L"}}
	s.clock = cptime.NewFixedClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	s.useCase = send_message_usecase.New(
		s.appender,
		fakeCountryChecker{known: cpcolls.NewSet("fr", "de")},
		s.authors,
		s.clock,
		send_message_usecase.Config{},
	)
}

func (s *testSuite) send(in send_message_usecase.In) (messages.Message, error) {
	return s.useCase.Execute(cpctx.AddIPToContext(context.Background(), "1.2.3.4"), in)
}

func validIn() send_message_usecase.In {
	return send_message_usecase.In{
		Account:    cpsession.NoAccount,
		AuthorName: "Bob",
		AuthorID:   "8f14e45f-ea23-4a1b-9c11-0b0d1a2b3c4d",
		CountryID:  "fr",
		Text:       "hello planet",
		UserAgent:  "curl/8",
	}
}

func (s *testSuite) TestNominalCase() {
	message, err := s.send(validIn())
	s.Require().NoError(err)

	s.NotEmpty(message.ID)
	s.Equal("guest_Bob", message.AuthorName)
	s.Equal("fr", message.CountryID)
	s.Equal("hello planet", message.Text)
	s.Equal(s.clock.Now(), message.SentAt)
	s.Equal("a1b2c3", message.AuthorTag)
	s.Equal([]string{"1.2.3.4"}, s.authors.ips, "the player module tags the address the message came from")

	s.Require().Len(s.appender.records, 1)
	s.Equal(message, s.appender.records[0].Message)
	s.Equal("8f14e45f-ea23-4a1b-9c11-0b0d1a2b3c4d", s.appender.records[0].AuthorID)
	s.Equal("curl/8", s.appender.records[0].UserAgent)
}

func (s *testSuite) TestTheSenderIPIsRecordedButNeverReturned() {
	message, err := s.send(validIn())
	s.Require().NoError(err)

	s.Require().Len(s.appender.records, 1)
	s.Equal("1.2.3.4", s.appender.records[0].IP)

	s.NotContains(message.AuthorTag, "1.2.3.4")
	s.NotContains(message.AuthorName, "1.2.3.4")
	s.NotContains(message.Text, "1.2.3.4")
}

func (s *testSuite) TestTheMessageIsCleanedBeforeItIsStored() {
	in := validIn()
	in.Text = "hello\tplanet"

	message, err := s.send(in)
	s.Require().NoError(err)
	s.Equal("hello planet", message.Text)
	s.Equal("hello planet", s.appender.records[0].Message.Text)
}

func (s *testSuite) TestAnInvalidTextIsRefusedAndNotStored() {
	in := validIn()
	in.Text = "   "

	_, err := s.send(in)
	s.ErrorIs(err, messages.ErrInvalidMessage)
	s.Empty(s.appender.records)
}

func (s *testSuite) TestAnInvalidNameIsRefusedAndNotStored() {
	in := validIn()
	in.AuthorName = ""

	_, err := s.send(in)
	s.ErrorIs(err, messages.ErrInvalidMessage)
	s.Empty(s.appender.records)
}

func (s *testSuite) TestInvalidCountryIsRefused() {
	in := validIn()
	in.CountryID = "atlantis"

	_, err := s.send(in)
	s.ErrorIs(err, messages.ErrInvalidMessage)
	s.Empty(s.appender.records)
}

func (s *testSuite) TestStorageFailureFailsTheSend() {
	s.appender.err = errors.New("disk on fire")

	_, err := s.send(validIn())
	s.Require().Error(err)
	s.NotErrorIs(err, messages.ErrInvalidMessage)
}

func (s *testSuite) TestAPlayerWithAUsernamePostsUnderItAndTheTypedNameIsNotRead() {
	in := validIn()
	in.Account = ada
	in.AuthorName = ""

	message, err := s.send(in)

	s.Require().NoError(err, "the typed name is not even checked")
	s.Equal("Ada_L", message.AuthorName)
	s.Equal("Ada_L", s.appender.records[0].Message.AuthorName)
	s.Equal("a1b2c3", message.AuthorTag, "a player is tagged as a guest is")
	s.Equal([]messages.AccountID{ada}, s.authors.asked)
}

func (s *testSuite) TestAnAccountWithNoUsernamePostsAsAGuest() {
	in := validIn()
	in.Account = guest

	message, err := s.send(in)

	s.Require().NoError(err)
	s.Equal("guest_Bob", message.AuthorName)
}

func (s *testSuite) TestAnAuthorThatCannotBeReadRefusesThePost() {
	s.authors.err = errors.New("the player module is stuck")
	in := validIn()
	in.Account = ada

	_, err := s.send(in)

	s.ErrorIs(err, messages.ErrAuthorUnavailable)
	s.NotErrorIs(err, messages.ErrInvalidMessage)
	s.Empty(s.appender.records)
}

func (s *testSuite) TestNoAccountIsAGuestAndIsStillTagged() {
	message, err := s.send(validIn())

	s.Require().NoError(err)
	s.Equal("guest_Bob", message.AuthorName)
	s.Equal("a1b2c3", message.AuthorTag)
	s.Equal([]messages.AccountID{cpsession.NoAccount}, s.authors.asked)
}

func (s *testSuite) TestAGuestsNameIsCleanedBeforeItIsPrefixed() {
	in := validIn()
	in.AuthorName = "  Bob\tthe builder\n"

	message, err := s.send(in)

	s.Require().NoError(err)
	s.Equal("guest_Bob the builder", message.AuthorName)
}

func (s *testSuite) TestAnInvalidGuestNameIsRefusedForAnAccountWithNoUsername() {
	in := validIn()
	in.Account = guest
	in.AuthorName = ""

	_, err := s.send(in)

	s.ErrorIs(err, messages.ErrInvalidMessage)
	s.Empty(s.appender.records)
}

type fakeAuthors struct {
	mu    sync.Mutex
	names map[messages.AccountID]string
	err   error
	asked []messages.AccountID
	ips   []string
}

func (f *fakeAuthors) Author(_ context.Context, account messages.AccountID, ip string) (messages.Author, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.asked = append(f.asked, account)
	if f.err != nil {
		return messages.Author{}, f.err
	}
	f.ips = append(f.ips, ip)
	return messages.Author{Username: f.names[account], Tag: "a1b2c3"}, nil
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

type fakeCountryChecker struct {
	known *cpcolls.Set[string]
}

func (f fakeCountryChecker) CheckCountry(country string) bool {
	return f.known.Contains(country)
}

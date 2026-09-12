package chat_service_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/domain/chat_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/stretchr/testify/suite"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	storage *fakeStorage
	clock   *cptime.FixedClock
	service *chat_service.Service
}

func (s *testSuite) SetupTest() {
	s.storage = &fakeStorage{}
	s.clock = cptime.NewFixedClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	s.service = s.newService(chat_service.Config{TagSalt: "pepper"})
}

func (s *testSuite) newService(config chat_service.Config) *chat_service.Service {
	return chat_service.New(
		s.storage,
		fakeCountryChecker{known: map[string]bool{"fr": true, "de": true}},
		s.clock,
		config,
	)
}

func (s *testSuite) post(req chat_service.PostRequest) (domain.ChatMessage, error) {
	return s.service.Post(s.ctxFromIP("1.2.3.4"), req)
}

func (s *testSuite) ctxFromIP(ip string) context.Context {
	return cpctx.AddIPToContext(context.Background(), ip)
}

func validRequest() chat_service.PostRequest {
	return chat_service.PostRequest{
		AuthorName: "Bob",
		AuthorID:   "8f14e45f-ea23-4a1b-9c11-0b0d1a2b3c4d",
		CountryID:  "fr",
		Text:       "hello planet",
	}
}

func (s *testSuite) TestNominalCase() {
	message, err := s.post(validRequest())
	s.Require().NoError(err)

	s.NotEmpty(message.ID)
	s.Equal("Bob", message.AuthorName)
	s.Equal("fr", message.CountryID)
	s.Equal("hello planet", message.Text)
	s.Equal(s.clock.Now(), message.SentAt)
	s.Len(s.storage.records, 1)
}

func (s *testSuite) TestTheSenderIPIsRecordedButNeverReturned() {
	message, err := s.post(validRequest())
	s.Require().NoError(err)

	s.Require().Len(s.storage.records, 1)
	s.Equal("1.2.3.4", s.storage.records[0].IP)

	s.NotContains(message.AuthorTag, "1.2.3.4")
	s.NotContains(message.AuthorName, "1.2.3.4")
	s.NotContains(message.Text, "1.2.3.4")
}

func (s *testSuite) TestTheSameSenderAlwaysGetsTheSameTag() {
	first, err := s.post(validRequest())
	s.Require().NoError(err)

	second, err := s.post(validRequest())
	s.Require().NoError(err)

	s.Equal(first.AuthorTag, second.AuthorTag)
}

func (s *testSuite) TestDifferentSendersGetDifferentTags() {
	mine, err := s.service.Post(s.ctxFromIP("1.2.3.4"), validRequest())
	s.Require().NoError(err)

	theirs, err := s.service.Post(s.ctxFromIP("5.6.7.8"), validRequest())
	s.Require().NoError(err)

	s.NotEqual(mine.AuthorTag, theirs.AuthorTag)
}

func (s *testSuite) TestTheSaltChangesTheTag() {
	otherwiseSalted := s.newService(chat_service.Config{TagSalt: "other pepper"})

	salted, err := s.post(validRequest())
	s.Require().NoError(err)

	other, err := otherwiseSalted.Post(s.ctxFromIP("1.2.3.4"), validRequest())
	s.Require().NoError(err)

	s.NotEqual(salted.AuthorTag, other.AuthorTag)
}

func (s *testSuite) TestEmptyTextIsRefused() {
	req := validRequest()
	req.Text = "   "

	_, err := s.post(req)
	s.ErrorIs(err, domain.ErrInvalidMessage)
}

func (s *testSuite) TestOverlongTextIsRefused() {
	req := validRequest()
	req.Text = strings.Repeat("a", 281)

	_, err := s.post(req)
	s.ErrorIs(err, domain.ErrInvalidMessage)
}

func (s *testSuite) TestLengthIsCountedInRunesNotBytes() {
	req := validRequest()
	req.Text = strings.Repeat("🌍", 280)

	_, err := s.post(req)
	s.NoError(err)
}

func (s *testSuite) TestInvalidUTF8IsRefused() {
	req := validRequest()
	req.Text = string([]byte{0xff, 0xfe, 0xfd})

	_, err := s.post(req)
	s.ErrorIs(err, domain.ErrInvalidMessage)
}

func (s *testSuite) TestControlCharactersAreStripped() {
	req := validRequest()
	req.Text = "hello\n{\"ip\":\"forged\"}\r\x00 planet"

	message, err := s.post(req)
	s.Require().NoError(err)

	s.NotContains(message.Text, "\n")
	s.NotContains(message.Text, "\r")
	s.NotContains(message.Text, "\x00")
}

func (s *testSuite) TestTabsBecomeSpaces() {
	req := validRequest()
	req.Text = "hello\tplanet"

	message, err := s.post(req)
	s.Require().NoError(err)
	s.Equal("hello planet", message.Text)
}

func (s *testSuite) TestEmptyNameIsRefused() {
	req := validRequest()
	req.AuthorName = ""

	_, err := s.post(req)
	s.ErrorIs(err, domain.ErrInvalidMessage)
}

func (s *testSuite) TestOverlongNameIsRefused() {
	req := validRequest()
	req.AuthorName = strings.Repeat("a", 25)

	_, err := s.post(req)
	s.ErrorIs(err, domain.ErrInvalidMessage)
}

func (s *testSuite) TestInvalidCountryIsRefused() {
	req := validRequest()
	req.CountryID = "atlantis"

	_, err := s.post(req)
	s.ErrorIs(err, domain.ErrInvalidMessage)
}

func (s *testSuite) TestOverlongAuthorIDIsTruncated() {
	req := validRequest()
	req.AuthorID = strings.Repeat("a", 5000)

	_, err := s.post(req)
	s.Require().NoError(err)

	s.Require().Len(s.storage.records, 1)
	s.LessOrEqual(len(s.storage.records[0].AuthorID), 64)
}

func (s *testSuite) TestOverlongUserAgentIsTruncated() {
	req := validRequest()
	req.UserAgent = strings.Repeat("a", 5000)

	_, err := s.post(req)
	s.Require().NoError(err)

	s.Require().Len(s.storage.records, 1)
	s.LessOrEqual(len(s.storage.records[0].UserAgent), 256)
}

func (s *testSuite) TestStorageFailureFailsThePost() {
	s.storage.err = errors.New("disk on fire")

	_, err := s.post(validRequest())
	s.Error(err)
}

func (s *testSuite) TestHistoryComesFromStorage() {
	_, err := s.post(validRequest())
	s.Require().NoError(err)

	s.Len(s.service.History(context.Background()), 1)
}

type fakeStorage struct {
	mu      sync.Mutex
	records []domain.ChatRecord
	err     error
}

func (f *fakeStorage) Append(_ context.Context, record domain.ChatRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.err != nil {
		return f.err
	}

	f.records = append(f.records, record)
	return nil
}

func (f *fakeStorage) History(_ context.Context) []domain.ChatMessage {
	f.mu.Lock()
	defer f.mu.Unlock()

	messages := make([]domain.ChatMessage, 0, len(f.records))
	for _, record := range f.records {
		messages = append(messages, record.Message)
	}
	return messages
}

type fakeCountryChecker struct {
	known map[string]bool
}

func (f fakeCountryChecker) CheckCountry(country string) bool {
	return f.known[country]
}

var (
	_ domain.Storage        = (*fakeStorage)(nil)
	_ domain.CountryChecker = fakeCountryChecker{}
	_ chat_service.IService = (*chat_service.Service)(nil)
)

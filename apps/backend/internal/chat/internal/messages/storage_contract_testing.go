//go:build testing

package messages

import (
	"context"
	"fmt"
	"time"

	"github.com/stretchr/testify/suite"
)

// StorageContractSuite is the behaviour every Storage shares. Embed it and set NewStorage.
type StorageContractSuite struct {
	suite.Suite

	// NewStorage builds an empty storage.
	NewStorage func() Storage

	storage Storage
}

var contractStart = time.Date(2024, 1, 1, 12, 0, 0, 123_456_000, time.UTC)

func (s *StorageContractSuite) SetupTest() {
	s.storage = s.NewStorage()
}

func contractRecord(text string, at time.Time) Record {
	return NewRecord(Message{
		ID:         MessageID("id-" + text),
		SentAt:     at,
		AuthorName: "Bob",
		CountryID:  "fr",
		Text:       text,
	}, "some-uuid", "203.0.113.7", "test-agent")
}

func (s *StorageContractSuite) append(texts ...string) {
	for i, text := range texts {
		s.Require().NoError(s.storage.Append(context.Background(),
			contractRecord(text, contractStart.Add(time.Duration(i)*time.Hour))))
	}
}

func (s *StorageContractSuite) recent(since time.Time, limit int) []string {
	recent, err := s.storage.Recent(context.Background(), since, limit)
	s.Require().NoError(err)

	texts := make([]string, 0, len(recent))
	for _, message := range recent {
		texts = append(texts, message.Text)
	}
	return texts
}

func (s *StorageContractSuite) shown(text string, since time.Time, limit int) bool {
	shown, err := s.storage.Shown(context.Background(), MessageID("id-"+text), since, limit)
	s.Require().NoError(err)
	return shown
}

func (s *StorageContractSuite) TestAnEmptyStorageHasNoRecentMessages() {
	s.Empty(s.recent(contractStart, 10))
}

func (s *StorageContractSuite) TestAMessageReadsBackAsItWasAppended() {
	admin := contractRecord("hello", contractStart)
	admin.Message.AuthorAdmin = true
	s.Require().NoError(s.storage.Append(context.Background(), admin))

	recent, err := s.storage.Recent(context.Background(), contractStart, 10)
	s.Require().NoError(err)

	s.Equal([]Message{admin.Message}, recent)
}

func (s *StorageContractSuite) TestRecentIsTheNewestWithinTheWindowOldestFirst() {
	texts := make([]string, 0, 5)
	for i := range 5 {
		texts = append(texts, fmt.Sprintf("msg-%d", i))
	}
	s.append(texts...)

	s.Equal([]string{"msg-3", "msg-4"}, s.recent(contractStart, 2))
	s.Equal([]string{"msg-2", "msg-3", "msg-4"}, s.recent(contractStart.Add(2*time.Hour), 10))
}

func (s *StorageContractSuite) TestRecentKeepsTheOrderMessagesWereAppendedIn() {
	s.Require().NoError(s.storage.Append(context.Background(), contractRecord("first", contractStart.Add(time.Second))))
	s.Require().NoError(s.storage.Append(context.Background(), contractRecord("second", contractStart)))

	s.Equal([]string{"first", "second"}, s.recent(contractStart, 10))
}

func (s *StorageContractSuite) TestAMessageIsShownOnlyWhileRecentAnswersIt() {
	s.append("old", "middle", "new")

	s.True(s.shown("new", contractStart, 2))
	s.True(s.shown("middle", contractStart, 2))
	s.False(s.shown("old", contractStart, 2), "pushed out by newer ones")
	s.False(s.shown("middle", contractStart.Add(2*time.Hour), 10), "sent before the window")
	s.False(s.shown("never-sent", contractStart, 10))
}

func (s *StorageContractSuite) TestDeleteBeforeRemovesOnlyOlderMessages() {
	s.Require().NoError(s.storage.Append(context.Background(), contractRecord("ancient", contractStart)))
	s.Require().NoError(s.storage.Append(context.Background(), contractRecord("recent", contractStart.Add(48*time.Hour))))

	deleted, err := s.storage.DeleteBefore(context.Background(), contractStart.Add(24*time.Hour))
	s.Require().NoError(err)

	s.Equal(int64(1), deleted)
	s.Equal([]string{"recent"}, s.recent(contractStart, 10))
}

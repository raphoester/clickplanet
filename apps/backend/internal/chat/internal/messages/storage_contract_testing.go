//go:build testing

package messages

import (
	"context"
	"time"

	"github.com/stretchr/testify/suite"
)

type StorageContractSuite struct {
	suite.Suite

	NewStorage func() Storage

	storage Storage
}

var contractStart = time.Date(2024, 1, 1, 12, 0, 0, 123_456_000, time.UTC)

func (s *StorageContractSuite) SetupTest() {
	s.storage = s.NewStorage()
}

var contractAccount = AccountID{15: 1}

func contractRecord(text string, at time.Time) Record {
	return NewRecord(NewMessage(MessageID("id-"+text), at, contractAccount, "fr", text), "some-uuid", "203.0.113.7", "test-agent")
}

func (s *StorageContractSuite) append(texts ...string) {
	for i, text := range texts {
		s.Require().NoError(s.storage.Append(context.Background(),
			contractRecord(text, contractStart.Add(time.Duration(i)*time.Hour))))
	}
}

func (s *StorageContractSuite) shown(text string, since time.Time, limit int) bool {
	shown, err := s.storage.Shown(context.Background(), MessageID("id-"+text), since, limit)
	s.Require().NoError(err)
	return shown
}

func (s *StorageContractSuite) TestAnEmptyStorageShowsNothing() {
	s.False(s.shown("hello", contractStart, 10))
}

func (s *StorageContractSuite) TestAMessageAppendedIsShown() {
	s.append("hello")

	s.True(s.shown("hello", contractStart, 10))
}

func (s *StorageContractSuite) TestTheNewestIsTheLastAppendedWhateverItsTime() {
	s.Require().NoError(s.storage.Append(context.Background(), contractRecord("first", contractStart.Add(time.Second))))
	s.Require().NoError(s.storage.Append(context.Background(), contractRecord("second", contractStart)))

	s.True(s.shown("second", contractStart, 1))
	s.False(s.shown("first", contractStart, 1))
}

func (s *StorageContractSuite) TestAMessageIsShownOnlyWhileItIsAmongTheNewestInTheWindow() {
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
	s.True(s.shown("recent", contractStart, 10))
	s.False(s.shown("ancient", contractStart, 10))
}

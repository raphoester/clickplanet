//go:build testing

package announcements

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type StorageContractSuite struct {
	suite.Suite

	NewStorage func() Storage

	storage Storage
}

var contractStart = time.Date(2024, 1, 1, 12, 0, 0, 123_456_000, time.UTC)

func contractID(name string) AnnouncementID {
	return AnnouncementID(uuid.NewSHA1(uuid.NameSpaceOID, []byte(name)))
}

func (s *StorageContractSuite) SetupTest() {
	s.storage = s.NewStorage()
}

func contractAnnouncement(name string, at time.Time) Announcement {
	return Announcement{
		ID:      contractID(name),
		Kind:    KindBomb,
		At:      at,
		Payload: json.RawMessage(`{"country":"fr","ground":"de","tile":42,"cleared":3}`),
	}
}

func (s *StorageContractSuite) append(names ...string) {
	for i, name := range names {
		s.Require().NoError(s.storage.Append(context.Background(),
			contractAnnouncement(name, contractStart.Add(time.Duration(i)*time.Hour))))
	}
}

func (s *StorageContractSuite) TestAnIDIsKeptOnce() {
	s.append("boom")

	s.Error(s.storage.Append(context.Background(), contractAnnouncement("boom", contractStart.Add(time.Hour))))
}

func (s *StorageContractSuite) TestDeleteBeforeRemovesWhatIsOlderAndCountsIt() {
	s.append("a", "b", "c")

	deleted, err := s.storage.DeleteBefore(context.Background(), contractStart.Add(90*time.Minute))
	s.Require().NoError(err)

	s.Equal(int64(2), deleted)
	rest, err := s.storage.DeleteBefore(context.Background(), contractStart.Add(24*time.Hour))
	s.Require().NoError(err)
	s.Equal(int64(1), rest, "what was newer was kept")
}

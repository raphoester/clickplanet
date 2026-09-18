//go:build testing

package announcements

import (
	"context"
	"encoding/json"
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

func contractAnnouncement(name string, at time.Time) Announcement {
	return Announcement{
		ID:      AnnouncementID("id-" + name),
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

func (s *StorageContractSuite) recent(since time.Time, limit int) []AnnouncementID {
	recent, err := s.storage.Recent(context.Background(), since, limit)
	s.Require().NoError(err)

	ids := make([]AnnouncementID, 0, len(recent))
	for _, announcement := range recent {
		ids = append(ids, announcement.ID)
	}
	return ids
}

func (s *StorageContractSuite) TestAnEmptyStorageHasNoRecentAnnouncements() {
	s.Empty(s.recent(contractStart, 10))
}

func (s *StorageContractSuite) TestAnAnnouncementReadsBackAsItWasAppended() {
	s.append("boom")

	recent, err := s.storage.Recent(context.Background(), contractStart, 10)
	s.Require().NoError(err)
	s.Require().Len(recent, 1)

	want := contractAnnouncement("boom", contractStart)
	s.Equal(want.ID, recent[0].ID)
	s.Equal(want.Kind, recent[0].Kind)
	s.True(want.At.Equal(recent[0].At), "at %s, want %s", recent[0].At, want.At)
	s.JSONEq(string(want.Payload), string(recent[0].Payload))
}

func (s *StorageContractSuite) TestRecentIsTheNewestOldestFirst() {
	s.append("a", "b", "c")

	s.Equal([]AnnouncementID{"id-b", "id-c"}, s.recent(contractStart, 2))
}

func (s *StorageContractSuite) TestRecentLeavesOutWhatIsOlderThanSince() {
	s.append("a", "b", "c")

	s.Equal([]AnnouncementID{"id-c"}, s.recent(contractStart.Add(90*time.Minute), 10))
}

func (s *StorageContractSuite) TestDeleteBeforeRemovesWhatIsOlderAndCountsIt() {
	s.append("a", "b", "c")

	deleted, err := s.storage.DeleteBefore(context.Background(), contractStart.Add(90*time.Minute))
	s.Require().NoError(err)

	s.Equal(int64(2), deleted)
	s.Equal([]AnnouncementID{"id-c"}, s.recent(time.Time{}, 10))
}

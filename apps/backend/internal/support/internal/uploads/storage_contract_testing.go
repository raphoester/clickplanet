//go:build testing

package uploads

import (
	"errors"
	"sync"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type StorageContractSuite struct {
	suite.Suite

	NewStorage func() Storage
	// Makes a note name the files.
	Send func(storage Storage, ids ...notes.AttachmentID)

	storage Storage
}

var contractStart = time.Date(2026, 10, 14, 12, 0, 0, 123_456_000, time.UTC)

func (s *StorageContractSuite) SetupTest() {
	s.storage = s.NewStorage()
}

func (s *StorageContractSuite) upload(n byte, size int64, at time.Time) Upload {
	upload, err := NewUpload(notes.Attachment{ID: notes.AttachmentID{14: 7, 15: n}, MediaType: notes.MediaPNG, Size: size}, at)
	s.Require().NoError(err)
	return upload
}

func (s *StorageContractSuite) reserved(n byte, size int64, at time.Time) Upload {
	upload := s.upload(n, size, at)
	s.Require().NoError(s.storage.Reserve(s.T().Context(), upload, func(int64) error { return nil }))
	return upload
}

func (s *StorageContractSuite) held() int64 {
	bytes, err := s.storage.Held(s.T().Context())
	s.Require().NoError(err)
	return bytes
}

func (s *StorageContractSuite) unsent(cutoff time.Time, limit int) []Snapshot {
	found, err := s.storage.Unsent(s.T().Context(), cutoff, limit)
	s.Require().NoError(err)

	taken := make([]Snapshot, 0, len(found))
	for _, upload := range found {
		snapshot := upload.TakeSnapshot()
		snapshot.At = snapshot.At.UTC()
		taken = append(taken, snapshot)
	}
	return taken
}

func (s *StorageContractSuite) TestNothingIsHeldAtFirst() {
	s.Equal(int64(0), s.held())
	s.Empty(s.unsent(contractStart.Add(time.Hour), 10))
}

func (s *StorageContractSuite) TestAReservedUploadIsHeldAndUnsentUntilANoteNamesIt() {
	first := s.reserved(1, 400, contractStart)
	s.reserved(2, 600, contractStart.Add(time.Second))

	s.Equal(int64(1000), s.held())
	s.Len(s.unsent(contractStart.Add(time.Hour), 10), 2)

	s.Send(s.storage, first.TakeSnapshot().ID)

	s.Equal(int64(1000), s.held(), "a file a note names still takes its room")
	unsent := s.unsent(contractStart.Add(time.Hour), 10)
	s.Require().Len(unsent, 1)
	s.Equal(notes.AttachmentID{14: 7, 15: 2}, unsent[0].ID)
}

func (s *StorageContractSuite) TestTheCheckIsToldTheBytesHeldAndARefusalKeepsNothing() {
	s.reserved(1, 400, contractStart)
	refused := errors.New("no room")

	var told int64
	err := s.storage.Reserve(s.T().Context(), s.upload(2, 700, contractStart), func(held int64) error {
		told = held
		return refused
	})

	s.Require().ErrorIs(err, refused)
	s.Equal(int64(400), told)
	s.Equal(int64(400), s.held())
	s.Len(s.unsent(contractStart.Add(time.Hour), 10), 1)
}

func (s *StorageContractSuite) TestReservationsAtOnceNeverPassTheRoom() {
	const (
		asked    = 20
		size     = 10
		capacity = 100
	)
	var wait sync.WaitGroup
	for n := range asked {
		wait.Go(func() {
			_ = s.storage.Reserve(s.T().Context(), s.upload(byte(n+1), size, contractStart), func(held int64) error {
				if held+size > capacity {
					return errors.New("no room")
				}
				return nil
			})
		})
	}
	wait.Wait()

	s.Equal(int64(capacity), s.held())
}

func (s *StorageContractSuite) TestTheUnsentComeOldestFirstFromBeforeTheCutoffUpToTheLimit() {
	s.reserved(1, 1, contractStart.Add(2*time.Minute))
	s.reserved(2, 1, contractStart)
	s.reserved(3, 1, contractStart.Add(time.Minute))
	s.reserved(4, 1, contractStart.Add(time.Hour))

	unsent := s.unsent(contractStart.Add(time.Hour), 2)

	s.Require().Len(unsent, 2)
	s.Equal(notes.AttachmentID{14: 7, 15: 2}, unsent[0].ID)
	s.Equal(notes.AttachmentID{14: 7, 15: 3}, unsent[1].ID)
	s.Len(s.unsent(contractStart.Add(time.Hour), 10), 3, "an upload at the cutoff is not from before it")
}

func (s *StorageContractSuite) TestADiscardedUploadIsKeptNoMoreAndFreesItsRoom() {
	gone := s.reserved(1, 400, contractStart)
	s.reserved(2, 600, contractStart.Add(time.Second))

	s.Require().NoError(s.storage.Save(s.T().Context(), gone.Discarded()))

	s.Equal(int64(600), s.held())
	unsent := s.unsent(contractStart.Add(time.Hour), 10)
	s.Require().Len(unsent, 1)
	s.Equal(notes.AttachmentID{14: 7, 15: 2}, unsent[0].ID)
}

func (s *StorageContractSuite) TestDiscardingWhatIsNotKeptIsNotAnError() {
	s.Require().NoError(s.storage.Save(s.T().Context(), s.upload(1, 400, contractStart).Discarded()))
	s.Require().NoError(s.storage.Save(s.T().Context()))

	s.Equal(int64(0), s.held())
}

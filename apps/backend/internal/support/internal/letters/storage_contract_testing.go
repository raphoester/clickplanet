//go:build testing

package letters

import (
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

// StorageContractSuite is what every Storage keeps.
type StorageContractSuite struct {
	suite.Suite
	NewStorage func() Storage

	storage Storage
}

var (
	writtenAt = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	ada       = notes.AccountID{15: 1}
	bob       = notes.AccountID{15: 2}
)

func (s *StorageContractSuite) SetupTest() {
	s.storage = s.NewStorage()
}

func (s *StorageContractSuite) kept(snapshots ...Snapshot) {
	for _, snapshot := range snapshots {
		letter, err := snapshot.Restore()
		s.Require().NoError(err)
		s.Require().NoError(s.storage.Save(s.T().Context(), letter))
	}
}

func snapshotsOf(found []Letter) []Snapshot {
	snapshots := make([]Snapshot, 0, len(found))
	for _, letter := range found {
		snapshots = append(snapshots, letter.TakeSnapshot())
	}
	return snapshots
}

func (s *StorageContractSuite) TestALetterIsReadAsItWasKept() {
	written := TestWritten(LetterID{15: 1}, ada, writtenAt)
	s.kept(written)

	kept, err := s.storage.Letter(s.T().Context(), LetterID{15: 1})
	s.Require().NoError(err)
	s.Equal(written, kept.TakeSnapshot())
}

func (s *StorageContractSuite) TestALetterNeverKeptIsUnknown() {
	_, err := s.storage.Letter(s.T().Context(), LetterID{15: 1})
	s.Require().ErrorIs(err, ErrUnknownLetter)
}

func (s *StorageContractSuite) TestALetterKeptAgainKeepsItsMailsRepliesAndClosing() {
	written := TestWritten(LetterID{15: 1}, ada, writtenAt)
	written.Sent = append(written.Sent, Sent{Text: "Thanks!", Mail: "l2@clickplanet.lol", At: writtenAt.Add(2 * time.Hour)})
	written.Replies = []conversations.Reply{{
		Mail: "r1@mail.example.com", From: "ada@example.com", Text: "Barcelona.", At: writtenAt.Add(time.Hour),
	}}
	written.ClosedAt = writtenAt.Add(3 * time.Hour)
	s.kept(written)

	kept, err := s.storage.Letter(s.T().Context(), LetterID{15: 1})
	s.Require().NoError(err)
	s.Equal(written, kept.TakeSnapshot())
}

func (s *StorageContractSuite) TestAnAnonymizedLetterKeepsWhatWasSaid() {
	written := TestWritten(LetterID{15: 1}, ada, writtenAt)
	written.Replies = []conversations.Reply{{
		Mail: "r1@mail.example.com", From: "ada@example.com", Text: "Barcelona.", At: writtenAt.Add(time.Hour),
	}}
	s.kept(written)
	letter, err := s.storage.Letter(s.T().Context(), LetterID{15: 1})
	s.Require().NoError(err)
	s.Require().NoError(s.storage.Save(s.T().Context(), letter.Anonymized()))

	kept, err := s.storage.Letter(s.T().Context(), LetterID{15: 1})
	s.Require().NoError(err)
	s.Equal(letter.Anonymized().TakeSnapshot(), kept.TakeSnapshot())
	to, err := s.storage.To(s.T().Context(), ada)
	s.Require().NoError(err)
	s.Empty(to)
}

func (s *StorageContractSuite) TestToIsTheAccountsLettersTheFirstWrittenFirst() {
	later := TestWritten(LetterID{15: 1}, ada, writtenAt.Add(time.Hour))
	first := TestWritten(LetterID{15: 2}, ada, writtenAt)
	s.kept(later, first, TestWritten(LetterID{15: 3}, bob, writtenAt))

	to, err := s.storage.To(s.T().Context(), ada)
	s.Require().NoError(err)
	s.Equal([]Snapshot{first, later}, snapshotsOf(to))
}

func (s *StorageContractSuite) TestAboutIsTheLetterAboutTheSubmission() {
	catalonia := TestWritten(LetterID{15: 1}, ada, writtenAt)
	s.kept(catalonia, TestWritten(LetterID{15: 2}, ada, writtenAt))

	about, err := s.storage.About(s.T().Context(), catalonia.Submission)
	s.Require().NoError(err)
	s.Equal([]Snapshot{catalonia}, snapshotsOf(about))
	none, err := s.storage.About(s.T().Context(), SubmissionID{15: 9})
	s.Require().NoError(err)
	s.Empty(none)
}

func (s *StorageContractSuite) TestASecondLetterAboutASubmissionIsRefused() {
	first := TestWritten(LetterID{15: 1}, ada, writtenAt)
	s.kept(first)
	second := TestWritten(LetterID{15: 2}, ada, writtenAt.Add(time.Hour))
	second.Submission = first.Submission

	err := s.storage.Save(s.T().Context(), Restored(second))
	s.Require().ErrorIs(err, ErrWrittenAlready)
	about, err := s.storage.About(s.T().Context(), first.Submission)
	s.Require().NoError(err)
	s.Equal([]Snapshot{first}, snapshotsOf(about))
}

func (s *StorageContractSuite) TestWrittenBeforeIsTheLettersThatStillNameTheirPlayer() {
	old := TestWritten(LetterID{15: 1}, ada, writtenAt)
	anonymized := TestWritten(LetterID{15: 2}, notes.AccountID{}, writtenAt)
	anonymized.To = ""
	s.kept(old, anonymized, TestWritten(LetterID{15: 3}, bob, writtenAt.Add(time.Hour)))

	before, err := s.storage.WrittenBefore(s.T().Context(), writtenAt.Add(time.Minute))
	s.Require().NoError(err)
	s.Equal([]Snapshot{old}, snapshotsOf(before))
}

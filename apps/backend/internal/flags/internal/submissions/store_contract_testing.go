//go:build testing

package submissions

import (
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
)

// StoreContractSuite is what every Store keeps.
type StoreContractSuite struct {
	suite.Suite
	NewStore func() Store

	store Store
}

var (
	submittedAt = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	ada         = catalog.AccountID{15: 1}
	bob         = catalog.AccountID{15: 2}
)

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

func (s *StoreContractSuite) kept(snapshots ...Snapshot) []Submission {
	submissions := make([]Submission, 0, len(snapshots))
	for _, snapshot := range snapshots {
		submissions = append(submissions, Restored(snapshot))
	}
	s.Require().NoError(s.store.Save(s.T().Context(), submissions...))
	return submissions
}

func snapshotsOf(submissions []Submission) []Snapshot {
	snapshots := make([]Snapshot, 0, len(submissions))
	for _, submission := range submissions {
		snapshots = append(snapshots, submission.TakeSnapshot())
	}
	return snapshots
}

func (s *StoreContractSuite) TestASubmissionIsReadAsItWasKept() {
	waiting := TestWaiting(SubmissionID{15: 1}, ada, submittedAt)
	waiting.Pin = catalog.Pin{Latitude: -33.8688, Longitude: 151.2093}
	waiting.Anthem = "au"
	s.kept(waiting)

	kept, err := s.store.Submission(s.T().Context(), SubmissionID{15: 1})
	s.Require().NoError(err)
	s.Equal(waiting, kept.TakeSnapshot())
}

func (s *StoreContractSuite) TestASubmissionNeverKeptIsUnknown() {
	_, err := s.store.Submission(s.T().Context(), SubmissionID{15: 1})
	s.Require().ErrorIs(err, ErrUnknownSubmission)
}

func (s *StoreContractSuite) TestARejectionIsKeptWithItsReasonAndNote() {
	waiting := s.kept(TestWaiting(SubmissionID{15: 1}, ada, submittedAt))[0]
	rejected, err := waiting.Rejected(Rejection{Reason: Other, Note: "Rennes is a city: send Brittany's flag."},
		submittedAt.Add(time.Hour))
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), rejected))

	kept, err := s.store.Submission(s.T().Context(), SubmissionID{15: 1})
	s.Require().NoError(err)
	s.Equal(rejected.TakeSnapshot(), kept.TakeSnapshot())
}

func (s *StoreContractSuite) TestASubmissionKeptAgainKeepsWhatChanged() {
	waiting := s.kept(TestWaiting(SubmissionID{15: 1}, ada, submittedAt))[0]
	approved, err := waiting.Approved(catalog.NamesOf(nil), submittedAt.Add(time.Hour))
	s.Require().NoError(err)
	opened, _, err := approved.Open(catalog.NewOpening(&catalog.Sequence{}, fixed(submittedAt.Add(2*time.Hour))), catalog.NamesOf(nil),
		submittedAt.Add(30*24*time.Hour))
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), opened))

	kept, err := s.store.Submission(s.T().Context(), SubmissionID{15: 1})
	s.Require().NoError(err)
	s.Equal(opened.TakeSnapshot(), kept.TakeSnapshot())
	s.Equal(catalog.FlagID("r-0000000001"), kept.TakeSnapshot().Flag)
}

func (s *StoreContractSuite) TestFoundedIsTheAccountsSubmissionsTheNewestFirst() {
	first := TestWaiting(SubmissionID{15: 1}, ada, submittedAt)
	first.State, first.Rejection = Rejected, Rejection{Reason: Exists}
	second := TestWaiting(SubmissionID{15: 2}, ada, submittedAt.Add(time.Hour))
	s.kept(first, second, TestWaiting(SubmissionID{15: 3}, bob, submittedAt))

	founded, err := s.store.Founded(s.T().Context(), ada)
	s.Require().NoError(err)
	s.Equal([]Snapshot{second, first}, snapshotsOf(founded))
}

func (s *StoreContractSuite) TestRejectedHeldIsEveryRejectionWhosePictureIsStillHeld() {
	held := TestWaiting(SubmissionID{15: 1}, ada, submittedAt)
	held.State, held.Rejection = Rejected, Rejection{Reason: Hateful}
	deleted := TestWaiting(SubmissionID{15: 2}, bob, submittedAt)
	deleted.State, deleted.Rejection, deleted.Held = Rejected, Rejection{Reason: Hateful}, false
	approved := TestWaiting(SubmissionID{15: 3}, bob, submittedAt)
	approved.State, approved.Held = Approved, false
	s.kept(held, deleted, approved, TestWaiting(SubmissionID{15: 4}, ada, submittedAt))

	rejected, err := s.store.RejectedHeld(s.T().Context())
	s.Require().NoError(err)
	s.Equal([]Snapshot{held}, snapshotsOf(rejected))
}

func (s *StoreContractSuite) TestASubmissionThatWaitsIsDeleted() {
	s.kept(TestWaiting(SubmissionID{15: 1}, ada, submittedAt), TestWaiting(SubmissionID{15: 2}, bob, submittedAt))

	s.Require().NoError(s.store.DeleteWaiting(s.T().Context(), SubmissionID{15: 1}))

	_, err := s.store.Submission(s.T().Context(), SubmissionID{15: 1})
	s.Require().ErrorIs(err, ErrUnknownSubmission)
	_, err = s.store.Submission(s.T().Context(), SubmissionID{15: 2})
	s.Require().NoError(err, "another stays")
}

func (s *StoreContractSuite) TestADecidedSubmissionIsNeverDeleted() {
	rejected := TestWaiting(SubmissionID{15: 1}, ada, submittedAt)
	rejected.State, rejected.Rejection = Rejected, Rejection{Reason: Exists}
	s.kept(rejected)

	s.Require().ErrorIs(s.store.DeleteWaiting(s.T().Context(), SubmissionID{15: 1}), ErrNotWaiting)
	s.Require().ErrorIs(s.store.DeleteWaiting(s.T().Context(), SubmissionID{15: 3}), ErrNotWaiting, "one never kept")

	kept, err := s.store.Submission(s.T().Context(), SubmissionID{15: 1})
	s.Require().NoError(err)
	s.Equal(rejected, kept.TakeSnapshot())
}

func (s *StoreContractSuite) TestARejectionAppealedIsReadWaitingWithNoRejection() {
	waiting := s.kept(TestWaiting(SubmissionID{15: 1}, ada, submittedAt))[0]
	rejected, err := waiting.Rejected(Rejection{Reason: NotAPlace, Note: "A club is not a state."}, submittedAt)
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), rejected))
	reopened, err := rejected.Reopened(ada, nil, submittedAt.Add(time.Hour))
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), reopened))

	kept, err := s.store.Submission(s.T().Context(), SubmissionID{15: 1})
	s.Require().NoError(err)
	s.Equal(Waiting, kept.TakeSnapshot().State)
	s.Equal(Rejection{}, kept.TakeSnapshot().Rejection)
	s.Equal(submittedAt.Add(time.Hour), kept.TakeSnapshot().DecidedAt)
}

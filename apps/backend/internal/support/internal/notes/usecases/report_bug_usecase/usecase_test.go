package report_bug_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/inmemory_attachment_bucket"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/inmemory_note_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/report_bug_usecase"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

var (
	now   = time.Date(2026, 10, 11, 20, 0, 0, 0, time.UTC)
	ada   = notes.AccountID{15: 1}
	phone = notes.Diagnostics{
		Build: "play-a1b2c3.js", ViewportWidth: 390, ViewportHeight: 844, ConnectionDown: true, UserAgent: "Mozilla/5.0 (iPhone)",
	}
	rules = notes.AttachmentRules{MaxPerNote: 2, MaxBytes: 1000}
	files = []notes.Attachment{
		{ID: notes.AttachmentID{0: 0xa, 15: 1}, MediaType: notes.MediaPNG, Size: 400},
		{ID: notes.AttachmentID{0: 0xa, 15: 2}, MediaType: notes.MediaMP4, Size: 1000},
	}
)

type failingIDs struct{}

func (failingIDs) NewID() (notes.NoteID, error) { return notes.NoteID{}, errors.New("no entropy") }

type testSuite struct {
	suite.Suite

	notes   *inmemory_note_storage.Storage
	bucket  *inmemory_attachment_bucket.Bucket
	useCase *report_bug_usecase.UseCase
}

func (s *testSuite) SetupTest() {
	s.notes = inmemory_note_storage.New()
	s.bucket = inmemory_attachment_bucket.New()
	s.useCase = s.build(&notes.SequentialIDs{})
}

func (s *testSuite) build(ids notes.IDProvider) *report_bug_usecase.UseCase {
	return report_bug_usecase.New(ids, cptime.NewFixedClock(now), rules, s.notes, s.bucket)
}

func (s *testSuite) TestABugReportIsKeptCleanedWithAnIDTheTimeAndItsDiagnostics() {
	err := s.useCase.Execute(s.T().Context(),
		report_bug_usecase.In{Account: ada, Text: "  the globe is black  ", Diagnostics: phone})
	s.Require().NoError(err)

	s.Equal([]notes.Snapshot{{
		ID: notes.NoteID{15: 1}, Kind: notes.KindBug, Account: ada, Text: "the globe is black", Diagnostics: phone, At: now,
	}}, s.notes.Kept())
}

func (s *testSuite) TestABugReportIsKeptWithTheFilesTheBucketHoldsInTheOrderNamed() {
	s.bucket.Put(files...)

	err := s.useCase.Execute(s.T().Context(),
		report_bug_usecase.In{Account: ada, Text: "the globe is black", Attachments: []notes.AttachmentID{files[1].ID, files[0].ID}})
	s.Require().NoError(err)

	s.Equal([]notes.Snapshot{{
		ID: notes.NoteID{15: 1}, Kind: notes.KindBug, Account: ada, Text: "the globe is black",
		Attachments: []notes.Attachment{files[1], files[0]}, At: now,
	}}, s.notes.Kept())
}

func (s *testSuite) TestAFileTheBucketDoesNotHoldIsRefusedAndNothingIsKept() {
	s.bucket.Put(files[0])

	err := s.useCase.Execute(s.T().Context(),
		report_bug_usecase.In{Account: ada, Text: "the globe is black", Attachments: []notes.AttachmentID{files[0].ID, files[1].ID}})

	s.Require().ErrorIs(err, notes.ErrNotUploaded)
	s.Empty(s.notes.Kept())
}

func (s *testSuite) TestFilesTheRulesDoNotTakeAreRefusedAndNothingIsKept() {
	third := notes.Attachment{ID: notes.AttachmentID{0: 0xa, 15: 3}, MediaType: notes.MediaPNG, Size: 10}
	s.bucket.Put(append(append([]notes.Attachment{}, files...), third)...)

	err := s.useCase.Execute(s.T().Context(), report_bug_usecase.In{
		Account: ada, Text: "the globe is black", Attachments: []notes.AttachmentID{files[0].ID, files[1].ID, third.ID},
	})

	s.Require().ErrorIs(err, notes.ErrInvalidAttachment)
	s.Empty(s.notes.Kept())
}

func (s *testSuite) TestABugReportWhoseFilesCannotBeReadIsNotKept() {
	s.bucket.Put(files...)
	cause := errors.New("the bucket is down")
	s.bucket.FailWith(cause)

	err := s.useCase.Execute(s.T().Context(),
		report_bug_usecase.In{Account: ada, Text: "the globe is black", Attachments: []notes.AttachmentID{files[0].ID}})

	s.Require().ErrorIs(err, cause)
	s.Empty(s.notes.Kept())
}

func (s *testSuite) TestAnInvalidTextIsRefusedAndNothingIsKept() {
	err := s.useCase.Execute(s.T().Context(), report_bug_usecase.In{Account: ada, Text: "   ", Diagnostics: phone})

	s.Require().ErrorIs(err, notes.ErrInvalidText)
	s.Empty(s.notes.Kept())
}

func (s *testSuite) TestABugReportWithNoIDIsNotKept() {
	err := s.build(failingIDs{}).Execute(s.T().Context(), report_bug_usecase.In{Account: ada, Text: "the globe is black"})

	s.Require().Error(err)
	s.NotErrorIs(err, notes.ErrInvalidText)
	s.Empty(s.notes.Kept())
}

func (s *testSuite) TestABugReportThatCannotBeKeptFails() {
	cause := errors.New("postgres is down")
	s.notes.FailWith(cause)

	err := s.useCase.Execute(s.T().Context(), report_bug_usecase.In{Account: ada, Text: "the globe is black"})

	s.Require().ErrorIs(err, cause)
}

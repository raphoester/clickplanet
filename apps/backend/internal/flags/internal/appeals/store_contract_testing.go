//go:build testing

package appeals

import (
	"context"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

// StoreContractSuite is what every Store keeps. Seed keeps the submission an appeal is of, where a store needs it.
type StoreContractSuite struct {
	suite.Suite
	NewStore func() Store
	Seed     func(ctx context.Context, submission submissions.Snapshot) error

	store Store
}

var (
	askedAt = time.Date(2026, 11, 27, 9, 0, 0, 0, time.UTC)
	founder = catalog.AccountID{15: 1}
)

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

func (s *StoreContractSuite) asked(id submissions.SubmissionID) Appeal {
	s.Require().NoError(s.Seed(s.T().Context(), submissions.TestWaiting(id, founder, askedAt)))
	appeal, err := Snapshot{
		Submission: id, Account: founder, State: Asked, AskedAt: askedAt, DecidedAt: askedAt,
		Rejection: submissions.Rejection{Reason: submissions.NotAPlace, Note: "A club is not a state."},
	}.Restore()
	s.Require().NoError(err)
	return appeal
}

func (s *StoreContractSuite) TestAnAppealIsReadAsItWasKept() {
	appeal := s.asked(submissions.SubmissionID{15: 1})
	s.Require().NoError(s.store.Save(s.T().Context(), appeal))

	kept, err := s.store.Of(s.T().Context(), submissions.SubmissionID{15: 1})
	s.Require().NoError(err)
	s.Require().Len(kept, 1)
	s.Equal(appeal.TakeSnapshot(), kept[0].TakeSnapshot())
}

func (s *StoreContractSuite) TestASubmissionNeverAppealedHasNone() {
	kept, err := s.store.Of(s.T().Context(), submissions.SubmissionID{15: 1})
	s.Require().NoError(err)
	s.Empty(kept)
}

func (s *StoreContractSuite) TestAnAppealAnsweredIsKeptAnswered() {
	appeal := s.asked(submissions.SubmissionID{15: 1})
	s.Require().NoError(s.store.Save(s.T().Context(), appeal))
	answered, changed := appeal.Answered(askedAt.Add(time.Hour))
	s.Require().True(changed)
	s.Require().NoError(s.store.Save(s.T().Context(), answered))

	kept, err := s.store.Of(s.T().Context(), submissions.SubmissionID{15: 1})
	s.Require().NoError(err)
	s.Require().Len(kept, 1)
	s.Equal(answered.TakeSnapshot(), kept[0].TakeSnapshot())
}

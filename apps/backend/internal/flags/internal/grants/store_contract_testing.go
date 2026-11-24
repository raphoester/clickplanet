//go:build testing

package grants

import (
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

// StoreContractSuite is what every Store keeps. NewStore hands a store whose flags table holds Flag, and whose
// submissions hold Submission.
type StoreContractSuite struct {
	suite.Suite
	NewStore   func() Store
	Flag       catalog.FlagID
	Submission submissions.SubmissionID

	store Store
}

var (
	grantedAt   = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	takenBackAt = grantedAt.Add(time.Hour)
)

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

func (s *StoreContractSuite) granted(id GrantID) Grant {
	grant, err := Snapshot{
		ID: id, Granted: true, Flag: s.Flag, Added: 31 * 24 * time.Hour, GrantedAt: grantedAt,
	}.Restore()
	s.Require().NoError(err)
	return grant
}

func (s *StoreContractSuite) TestAPaymentNeverHeardIsUnheard() {
	grant, err := s.store.Grant(s.T().Context(), "stripe:in_1")
	s.Require().NoError(err)
	s.Equal(Unheard("stripe:in_1").TakeSnapshot(), grant.TakeSnapshot())
	s.False(grant.Heard())
}

func (s *StoreContractSuite) TestAGrantIsReadAsItWasKept() {
	grant := s.granted("stripe:in_1")
	s.Require().NoError(s.store.Save(s.T().Context(), grant))

	kept, err := s.store.Grant(s.T().Context(), "stripe:in_1")
	s.Require().NoError(err)
	s.Equal(grant.TakeSnapshot(), kept.TakeSnapshot())
	s.True(kept.Heard())
}

func (s *StoreContractSuite) TestAGrantTakenBackKeepsItsTime() {
	grant := s.granted("stripe:in_1")
	s.Require().NoError(s.store.Save(s.T().Context(), grant))
	taken, err := Snapshot{
		ID: "stripe:in_1", Granted: true, Flag: s.Flag, Added: 31 * 24 * time.Hour, GrantedAt: grantedAt,
		TakenBack: true, TakenBackAt: takenBackAt,
	}.Restore()
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), taken))

	kept, err := s.store.Grant(s.T().Context(), "stripe:in_1")
	s.Require().NoError(err)
	s.Equal(taken.TakeSnapshot(), kept.TakeSnapshot())
}

func (s *StoreContractSuite) TestAPaymentTakenBackBeforeItsGrantIsKeptWithNoTime() {
	taken, err := Snapshot{ID: "stripe:in_1", TakenBack: true, TakenBackAt: takenBackAt}.Restore()
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), taken))

	kept, err := s.store.Grant(s.T().Context(), "stripe:in_1")
	s.Require().NoError(err)
	s.Equal(taken.TakeSnapshot(), kept.TakeSnapshot())
	s.True(kept.Heard())
}

func (s *StoreContractSuite) TestASecondSaveChangesNothingKept() {
	grant := s.granted("stripe:in_1")
	s.Require().NoError(s.store.Save(s.T().Context(), grant))
	other, err := Snapshot{
		ID: "stripe:in_1", Granted: true, Flag: s.Flag, Added: time.Hour, GrantedAt: takenBackAt,
	}.Restore()
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), other))

	kept, err := s.store.Grant(s.T().Context(), "stripe:in_1")
	s.Require().NoError(err)
	s.Equal(grant.TakeSnapshot(), kept.TakeSnapshot())
}

func (s *StoreContractSuite) held(id GrantID, at time.Time) Snapshot {
	return Snapshot{ID: id, Held: true, Submission: s.Submission, Period: Month, HeldAt: at}
}

func (s *StoreContractSuite) TestATimeHeldIsReadAsItWasKept() {
	held, err := s.held("stripe:in_1", grantedAt).Restore()
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), held))

	kept, err := s.store.Grant(s.T().Context(), "stripe:in_1")
	s.Require().NoError(err)
	s.Equal(held.TakeSnapshot(), kept.TakeSnapshot())
	s.True(kept.Heard())
}

func (s *StoreContractSuite) TestTheTimeHeldForASubmissionIsWhatIsNeitherReleasedNorTakenBack() {
	for i, id := range []GrantID{"stripe:in_3", "stripe:in_1", "stripe:in_2", "stripe:in_4"} {
		held, err := s.held(id, grantedAt.Add(time.Duration(i)*time.Minute)).Restore()
		s.Require().NoError(err)
		s.Require().NoError(s.store.Save(s.T().Context(), held))
	}
	released := s.held("stripe:in_2", grantedAt.Add(2*time.Minute))
	released.Granted, released.Flag, released.Added, released.GrantedAt = true, s.Flag, 31*24*time.Hour, takenBackAt
	takenBack := s.held("stripe:in_4", grantedAt.Add(3*time.Minute))
	takenBack.TakenBack, takenBack.TakenBackAt = true, takenBackAt
	for _, snapshot := range []Snapshot{released, takenBack} {
		grant, err := snapshot.Restore()
		s.Require().NoError(err)
		s.Require().NoError(s.store.Save(s.T().Context(), grant))
	}

	held, err := s.store.Held(s.T().Context(), s.Submission)
	s.Require().NoError(err)
	ids := make([]GrantID, 0, len(held))
	for _, grant := range held {
		ids = append(ids, grant.TakeSnapshot().ID)
	}
	s.Equal([]GrantID{"stripe:in_3", "stripe:in_1"}, ids)

	none, err := s.store.Held(s.T().Context(), submissions.SubmissionID{1})
	s.Require().NoError(err)
	s.Empty(none)
}

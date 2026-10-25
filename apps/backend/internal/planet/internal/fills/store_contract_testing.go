//go:build testing

package fills

import (
	"time"

	"github.com/stretchr/testify/suite"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

var contractAt = time.Date(2026, 10, 31, 21, 0, 0, 123456000, time.UTC)

const contractAccount = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"

func (s *StoreContractSuite) save(kept Snapshot) {
	fill, err := kept.Restore()
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), fill))
}

func (s *StoreContractSuite) same(want Snapshot, got Fill) {
	s.T().Helper()

	kept := got.TakeSnapshot()
	s.True(want.StartedAt.Equal(kept.StartedAt), "started at %s, read %s", want.StartedAt, kept.StartedAt)
	s.True(want.FinishedAt.Equal(kept.FinishedAt), "finished at %s, read %s", want.FinishedAt, kept.FinishedAt)
	kept.StartedAt, kept.FinishedAt = want.StartedAt, want.FinishedAt
	s.Equal(want, kept)
}

func (s *StoreContractSuite) TestNothingIsKeptBeforeTheFirstFill() {
	fill, err := s.store.Fill(s.T().Context(), "season-0:finale:fill")
	s.Require().NoError(err)
	s.Equal(Fill{}, fill)

	unfinished, err := s.store.Unfinished(s.T().Context())
	s.Require().NoError(err)
	s.Equal(Fill{}, unfinished)
}

func (s *StoreContractSuite) TestAFillIsFoundByItsRequestAsItWasLastKept() {
	started := Snapshot{ID: "season-0:finale:fill", Source: SourceInternal, StartedAt: contractAt}
	s.save(started)
	done := Snapshot{
		ID: "season-0:finale:fill", Source: SourceInternal, StartedAt: contractAt,
		Reached: contractAccount, Filled: 1200, FinishedAt: contractAt.Add(time.Minute),
	}
	s.save(done)
	operator := Snapshot{ID: "a1b2", Source: SourceOperator, StartedAt: contractAt.Add(time.Hour)}
	s.save(operator)

	found, err := s.store.Fill(s.T().Context(), "season-0:finale:fill")
	s.Require().NoError(err)
	s.same(done, found)

	found, err = s.store.Fill(s.T().Context(), "a1b2")
	s.Require().NoError(err)
	s.same(operator, found)
}

func (s *StoreContractSuite) TestTheUnfinishedFillIsTheOneStartedFirstOfThoseRunning() {
	s.save(Snapshot{
		ID: "done", Source: SourceInternal, StartedAt: contractAt.Add(-time.Hour), FinishedAt: contractAt,
	})
	second := Snapshot{ID: "second", Source: SourceOperator, StartedAt: contractAt.Add(time.Minute)}
	s.save(second)
	first := Snapshot{ID: "first", Source: SourceInternal, StartedAt: contractAt, Reached: contractAccount, Filled: 500}
	s.save(first)

	unfinished, err := s.store.Unfinished(s.T().Context())
	s.Require().NoError(err)
	s.same(first, unfinished)
}

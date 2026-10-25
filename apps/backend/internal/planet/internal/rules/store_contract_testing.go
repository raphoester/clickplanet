//go:build testing

package rules

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

func (s *StoreContractSuite) save(kept Snapshot) {
	s.Require().NoError(s.store.Save(s.T().Context(), kept.Restore()))
}

func (s *StoreContractSuite) same(want Snapshot, got Change) {
	s.T().Helper()

	kept := got.TakeSnapshot()
	s.True(want.AppliedAt.Equal(kept.AppliedAt), "applied at %s, read %s", want.AppliedAt, kept.AppliedAt)
	kept.AppliedAt = want.AppliedAt
	s.Equal(want, kept)
}

func (s *StoreContractSuite) TestNothingIsKeptBeforeTheFirstChange() {
	change, err := s.store.Change(s.T().Context(), "season-0:finale")
	s.Require().NoError(err)
	s.Equal(Snapshot{}, change.TakeSnapshot())

	latest, err := s.store.Latest(s.T().Context())
	s.Require().NoError(err)
	s.Equal(Snapshot{}, latest.TakeSnapshot())
}

func (s *StoreContractSuite) TestAChangeIsFoundByItsRequestAsItWasKept() {
	internal := Snapshot{Request: "season-0:finale", Set: "rush", Source: SourceInternal, AppliedAt: contractAt}
	operator := Snapshot{Request: "a1b2", Set: Default, Source: SourceOperator, AppliedAt: contractAt.Add(time.Minute)}
	s.save(internal)
	s.save(operator)

	found, err := s.store.Change(s.T().Context(), "season-0:finale")
	s.Require().NoError(err)
	s.same(internal, found)

	found, err = s.store.Change(s.T().Context(), "a1b2")
	s.Require().NoError(err)
	s.same(operator, found)

	none, err := s.store.Change(s.T().Context(), "season-0:days")
	s.Require().NoError(err)
	s.Equal(Snapshot{}, none.TakeSnapshot())
}

func (s *StoreContractSuite) TestTheLatestIsTheChangeAppliedLast() {
	s.save(Snapshot{Request: "first", Set: "rush", Source: SourceInternal, AppliedAt: contractAt})
	last := Snapshot{Request: "last", Set: Default, Source: SourceOperator, AppliedAt: contractAt.Add(time.Hour)}
	s.save(last)
	s.save(Snapshot{Request: "earlier", Set: "rush", Source: SourceOperator, AppliedAt: contractAt.Add(-time.Hour)})

	latest, err := s.store.Latest(s.T().Context())
	s.Require().NoError(err)
	s.same(last, latest)
}

func (s *StoreContractSuite) TestTwoChangesAtOneInstantLeaveTheOneSavedLast() {
	s.save(Snapshot{Request: "first", Set: "rush", Source: SourceInternal, AppliedAt: contractAt})
	second := Snapshot{Request: "second", Set: Default, Source: SourceOperator, AppliedAt: contractAt}
	s.save(second)

	latest, err := s.store.Latest(s.T().Context())
	s.Require().NoError(err)
	s.same(second, latest)
}

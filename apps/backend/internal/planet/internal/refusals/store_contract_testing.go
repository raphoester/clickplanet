//go:build testing

package refusals

import (
	"sync"
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

var (
	contractAt = time.Date(2026, 10, 26, 12, 30, 0, 123456000, time.UTC)

	aScope    = Caller{Scope: "2001:db8:1:2::/64"}
	anAccount = Caller{Account: "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"}
)

func (s *StoreContractSuite) keep(caller Caller, kept Snapshot) {
	s.Require().NoError(s.store.Change(s.T().Context(), caller, func(Refusal) Refusal { return kept.Restore() }))
}

func (s *StoreContractSuite) kept(caller Caller) Snapshot {
	refusal, err := s.store.Refusal(s.T().Context(), caller)
	s.Require().NoError(err)
	return refusal.TakeSnapshot()
}

func (s *StoreContractSuite) same(want, got Snapshot) {
	s.True(want.Until.Equal(got.Until), "until: want %s, got %s", want.Until, got.Until)
	s.True(want.DecidedAt.Equal(got.DecidedAt), "decided at: want %s, got %s", want.DecidedAt, got.DecidedAt)
}

func (s *StoreContractSuite) TestACallerNeverDecidedOnIsRefusedNothing() {
	s.Equal(Snapshot{}, s.kept(aScope))
	s.Equal(Snapshot{}, s.kept(anAccount))
}

func (s *StoreContractSuite) TestAChangeIsKeptAndReadBack() {
	s.keep(aScope, Snapshot{Until: contractAt.Add(time.Hour), DecidedAt: contractAt})

	s.same(Snapshot{Until: contractAt.Add(time.Hour), DecidedAt: contractAt}, s.kept(aScope))
	s.Equal(Snapshot{}, s.kept(anAccount))
}

func (s *StoreContractSuite) TestAScopeAndAnAccountOfTheSameNameAreTwoCallers() {
	scope, account := Caller{Scope: "same"}, Caller{Account: "same"}

	s.keep(scope, Snapshot{Until: contractAt.Add(time.Hour), DecidedAt: contractAt})

	s.Equal(Snapshot{}, s.kept(account))
}

func (s *StoreContractSuite) TestAChangeReplacesWhatWasKept() {
	s.keep(anAccount, Snapshot{Until: contractAt.Add(time.Hour), DecidedAt: contractAt})

	s.keep(anAccount, Snapshot{Until: contractAt.Add(time.Minute), DecidedAt: contractAt.Add(time.Minute)})

	s.same(Snapshot{Until: contractAt.Add(time.Minute), DecidedAt: contractAt.Add(time.Minute)}, s.kept(anAccount))
}

func (s *StoreContractSuite) TestAChangeIsHandedWhatWasKept() {
	s.keep(aScope, Snapshot{Until: contractAt.Add(time.Hour), DecidedAt: contractAt})

	var seen Snapshot
	s.Require().NoError(s.store.Change(s.T().Context(), aScope, func(refusal Refusal) Refusal {
		seen = refusal.TakeSnapshot()
		return refusal
	}))

	s.same(Snapshot{Until: contractAt.Add(time.Hour), DecidedAt: contractAt}, seen)
}

func (s *StoreContractSuite) TestChangesToOneCallerNeverOverlap() {
	const changes = 20
	s.keep(aScope, Snapshot{Until: contractAt, DecidedAt: contractAt})

	var wg sync.WaitGroup
	for range changes {
		wg.Go(func() {
			s.NoError(s.store.Change(s.T().Context(), aScope, func(refusal Refusal) Refusal {
				kept := refusal.TakeSnapshot()
				return Snapshot{Until: kept.Until.Add(time.Second), DecidedAt: kept.DecidedAt}.Restore()
			}))
		})
	}
	wg.Wait()

	s.True(contractAt.Add(changes*time.Second).Equal(s.kept(aScope).Until), "each change saw the one before it")
}

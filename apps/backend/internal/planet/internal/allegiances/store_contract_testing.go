//go:build testing

package allegiances

import (
	"errors"
	"sync"
	"time"

	"github.com/stretchr/testify/suite"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store
	Account  func(n int) Account

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

var contractAt = time.Date(2026, 10, 23, 12, 30, 0, 123456000, time.UTC)

func (s *StoreContractSuite) keep(account Account, kept Snapshot) {
	s.Require().NoError(s.store.Change(s.T().Context(), account, func(Allegiance) (Allegiance, error) {
		return kept.Restore(), nil
	}))
}

func (s *StoreContractSuite) kept(account Account) Snapshot {
	var seen Snapshot
	s.Require().NoError(s.store.Change(s.T().Context(), account, func(allegiance Allegiance) (Allegiance, error) {
		seen = allegiance.TakeSnapshot()
		return allegiance, nil
	}))
	return seen
}

func (s *StoreContractSuite) TestAnAccountNeverChangedStartsFromNothing() {
	s.Equal(Snapshot{}, s.kept(s.Account(1)))
}

func (s *StoreContractSuite) TestAChangeIsKeptAndSeenByTheNext() {
	s.keep(s.Account(1), Snapshot{Flag: "fr"})
	s.keep(s.Account(2), Snapshot{Flag: "de", ChangedAt: contractAt})

	s.Equal(Snapshot{Flag: "fr"}, s.kept(s.Account(1)), "a first flag keeps no time")
	s.True(contractAt.Equal(s.kept(s.Account(2)).ChangedAt))
	s.Equal("de", s.kept(s.Account(2)).Flag)
}

func (s *StoreContractSuite) TestAnAllegianceIsReadAsItWasKept() {
	s.keep(s.Account(1), Snapshot{Flag: "de", ChangedAt: contractAt})

	read, err := s.store.Allegiance(s.T().Context(), s.Account(1))
	s.Require().NoError(err)
	s.Equal("de", read.TakeSnapshot().Flag)
	s.True(contractAt.Equal(read.TakeSnapshot().ChangedAt))

	none, err := s.store.Allegiance(s.T().Context(), s.Account(2))
	s.Require().NoError(err)
	s.Equal(Snapshot{}, none.TakeSnapshot())
}

func (s *StoreContractSuite) TestAChangeReplacesWhatWasKept() {
	s.keep(s.Account(1), Snapshot{Flag: "fr"})

	s.keep(s.Account(1), Snapshot{Flag: "de", ChangedAt: contractAt})

	s.Equal("de", s.kept(s.Account(1)).Flag)
	s.True(contractAt.Equal(s.kept(s.Account(1)).ChangedAt))
}

func (s *StoreContractSuite) TestARefusedChangeKeepsNothingAndSaysWhy() {
	s.keep(s.Account(1), Snapshot{Flag: "fr"})
	refused := errors.New("refused")

	err := s.store.Change(s.T().Context(), s.Account(1), func(Allegiance) (Allegiance, error) {
		return Snapshot{Flag: "de"}.Restore(), refused
	})

	s.Require().ErrorIs(err, refused)
	s.Equal(Snapshot{Flag: "fr"}, s.kept(s.Account(1)))
}

func (s *StoreContractSuite) TestChangesToOneAccountNeverOverlap() {
	const changes = 20
	s.keep(s.Account(1), Snapshot{Flag: "fr", ChangedAt: contractAt})

	var wg sync.WaitGroup
	for range changes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.NoError(s.store.Change(s.T().Context(), s.Account(1), func(allegiance Allegiance) (Allegiance, error) {
				kept := allegiance.TakeSnapshot()
				return Snapshot{Flag: kept.Flag, ChangedAt: kept.ChangedAt.Add(time.Second)}.Restore(), nil
			}))
		}()
	}
	wg.Wait()

	s.True(contractAt.Add(changes*time.Second).Equal(s.kept(s.Account(1)).ChangedAt), "each change saw the one before it")
}

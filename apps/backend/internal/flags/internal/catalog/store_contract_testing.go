//go:build testing

package catalog

import (
	"time"

	"github.com/stretchr/testify/suite"
)

// StoreContractSuite is what every Store keeps.
type StoreContractSuite struct {
	suite.Suite
	NewStore func() Store

	store Store
}

var (
	openedAt = time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	closedAt = openedAt.Add(time.Hour)
)

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

func (s *StoreContractSuite) opened(id FlagID, name Name) Flag {
	flag := OpenFlag(id, name, openedAt)
	s.Require().NoError(s.store.Save(s.T().Context(), flag))
	return flag
}

func (s *StoreContractSuite) TestAFlagIsReadAsItWasKept() {
	flag := s.opened("r-0000000001", "Rennes")

	kept, err := s.store.Flag(s.T().Context(), "r-0000000001")
	s.Require().NoError(err)
	s.Equal(flag.TakeSnapshot(), kept.TakeSnapshot())
}

func (s *StoreContractSuite) TestAFlagNeverOpenedIsUnknown() {
	_, err := s.store.Flag(s.T().Context(), "r-0000000001")
	s.Require().ErrorIs(err, ErrUnknownFlag)
}

func (s *StoreContractSuite) TestAFlagKeptAgainKeepsWhatChanged() {
	flag := s.opened("r-0000000001", "Rennes")
	closed, err := flag.Closed(closedAt)
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), closed))

	kept, err := s.store.Flag(s.T().Context(), "r-0000000001")
	s.Require().NoError(err)
	s.Equal(Snapshot{ID: "r-0000000001", Name: "Rennes", State: Closed, DecidedAt: closedAt}, kept.TakeSnapshot())
}

func (s *StoreContractSuite) TestFlagsAreEveryFlagKeptTheClosedOnesToo() {
	rennes := s.opened("r-0000000001", "Rennes")
	closed, err := s.opened("r-0000000002", "Bavaria").Closed(closedAt)
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), closed))

	flags, err := s.store.Flags(s.T().Context())
	s.Require().NoError(err)
	snapshots := make([]Snapshot, 0, len(flags))
	for _, flag := range flags {
		snapshots = append(snapshots, flag.TakeSnapshot())
	}
	s.ElementsMatch([]Snapshot{rennes.TakeSnapshot(), closed.TakeSnapshot()}, snapshots)
}

func (s *StoreContractSuite) TestNoFlagIsKeptAtFirst() {
	flags, err := s.store.Flags(s.T().Context())
	s.Require().NoError(err)
	s.Empty(flags)
}

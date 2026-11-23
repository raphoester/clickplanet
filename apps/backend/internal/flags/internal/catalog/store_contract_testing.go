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
	s.Equal(closed.TakeSnapshot(), kept.TakeSnapshot())
	s.Equal(Closed, kept.TakeSnapshot().State)
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

func (s *StoreContractSuite) TestEndingIsTheOpenFlagsWhoseEndPassed() {
	rennes := s.opened("r-0000000001", "Rennes")
	s.opened("r-0000000002", "Bavaria")
	closed, err := OpenFlag("r-0000000003", "Corsica", openedAt.Add(-time.Hour)).Closed(closedAt)
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), closed))
	days, err := ParseDays(1)
	s.Require().NoError(err)
	later, err := rennes.TimeAdded(days, openedAt)
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), later))

	ending, err := s.store.Ending(s.T().Context(), openedAt.Add(30*24*time.Hour))
	s.Require().NoError(err)
	s.Require().Len(ending, 1)
	s.Equal(FlagID("r-0000000002"), ending[0].TakeSnapshot().ID, "not the one with a day more, nor the one closed")

	none, err := s.store.Ending(s.T().Context(), openedAt.Add(29*24*time.Hour))
	s.Require().NoError(err)
	s.Empty(none)
}

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
	removed, err := flag.Removed("Broke the rules.", closedAt)
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), removed))

	kept, err := s.store.Flag(s.T().Context(), "r-0000000001")
	s.Require().NoError(err)
	s.Equal(removed.TakeSnapshot(), kept.TakeSnapshot())
	s.Equal(Removed, kept.TakeSnapshot().State)
}

func (s *StoreContractSuite) TestAFlagsHolderIsKeptAndChanges() {
	held := HeldFlag("r-0000000001", "Rennes", TestHolder, openedAt)
	s.Require().NoError(s.store.Save(s.T().Context(), held))
	kept, err := s.store.Flag(s.T().Context(), "r-0000000001")
	s.Require().NoError(err)
	s.Equal(held.TakeSnapshot(), kept.TakeSnapshot())

	ended, ok := kept.Expired(openedAt.AddDate(0, 2, 0))
	s.Require().True(ok)
	claimed, _, err := ended.Paid(AccountID{15: 2}, openedAt.AddDate(0, 2, 1), Days(7), openedAt.AddDate(0, 2, 1))
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), claimed))

	kept, err = s.store.Flag(s.T().Context(), "r-0000000001")
	s.Require().NoError(err)
	s.Equal(AccountID{15: 2}, kept.TakeSnapshot().Holder)
}

func (s *StoreContractSuite) TestARemovalsNoteIsKeptUntilTheFlagComesBack() {
	removed, err := s.opened("r-0000000001", "Rennes").Removed("Broke the rules.", closedAt)
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), removed))

	kept, err := s.store.Flag(s.T().Context(), "r-0000000001")
	s.Require().NoError(err)
	s.Equal(Note("Broke the rules."), kept.TakeSnapshot().Note)

	days, err := ParseDays(7)
	s.Require().NoError(err)
	back, err := kept.TimeAdded(days, NamesOf(nil), closedAt.Add(time.Hour))
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), back))

	flags, err := s.store.Flags(s.T().Context())
	s.Require().NoError(err)
	s.Require().Len(flags, 1)
	s.Equal(back.TakeSnapshot(), flags[0].TakeSnapshot())
	s.Empty(flags[0].TakeSnapshot().Note)
}

func (s *StoreContractSuite) TestFlagsAreEveryFlagKeptTheClosedOnesToo() {
	rennes := s.opened("r-0000000001", "Rennes")
	closed, ok := s.opened("r-0000000002", "Bavaria").Expired(openedAt.AddDate(0, 2, 0))
	s.Require().True(ok)
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
	removed, err := OpenFlag("r-0000000003", "Corsica", openedAt.Add(-time.Hour)).Removed("Broke the rules.", closedAt)
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), removed))
	days, err := ParseDays(1)
	s.Require().NoError(err)
	later, err := rennes.TimeAdded(days, NamesOf(nil), openedAt)
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), later))

	ending, err := s.store.Ending(s.T().Context(), openedAt.Add(30*24*time.Hour))
	s.Require().NoError(err)
	s.Require().Len(ending, 1)
	s.Equal(FlagID("r-0000000002"), ending[0].TakeSnapshot().ID, "not the one with a day more, nor the one removed")

	none, err := s.store.Ending(s.T().Context(), openedAt.Add(29*24*time.Hour))
	s.Require().NoError(err)
	s.Empty(none)
}

func (s *StoreContractSuite) TestTwoFlagsNeverShareANameWhateverItsCase() {
	s.opened("r-0000000001", "Rennes")

	err := s.store.Save(s.T().Context(), OpenFlag("r-0000000002", "RENNES", openedAt))

	s.Require().ErrorIs(err, ErrNameTaken)
	flags, err := s.store.Flags(s.T().Context())
	s.Require().NoError(err)
	s.Len(flags, 1)
}

func (s *StoreContractSuite) TestARemovedFlagsNameIsFreeForANewFlag() {
	removed, err := s.opened("r-0000000001", "Rennes").Removed("Broke the rules.", closedAt)
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), removed))

	s.Require().NoError(s.store.Save(s.T().Context(), OpenFlag("r-0000000002", "RENNES", closedAt)))

	back, err := removed.TimeAdded(Days(7), NamesOf(nil), closedAt)
	s.Require().NoError(err, "the names read say nothing of the new flag")
	s.Require().ErrorIs(s.store.Save(s.T().Context(), back), ErrNameTaken, "two open flags never share a name")
}

func (s *StoreContractSuite) TestAFlagClosedAtItsEndKeepsItsName() {
	closed, ok := s.opened("r-0000000001", "Rennes").Expired(openedAt.AddDate(0, 2, 0))
	s.Require().True(ok)
	s.Require().NoError(s.store.Save(s.T().Context(), closed))

	err := s.store.Save(s.T().Context(), OpenFlag("r-0000000002", "rennes", closedAt))

	s.Require().ErrorIs(err, ErrNameTaken)
}

//go:build testing

package players

import (
	"time"

	"github.com/stretchr/testify/suite"
)

// StoreContractSuite is the behaviour every Store shares. Embed it and set NewStore.
type StoreContractSuite struct {
	suite.Suite

	// NewStore answers an empty store.
	NewStore func() Store

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

var contractAt = time.Date(2026, 9, 17, 23, 30, 0, 123_456_000, time.UTC)

func contractProfile(account byte, name Name) Profile {
	return Profile{Account: AccountID{15: account}, Name: name, UpdatedAt: contractAt}
}

func (s *StoreContractSuite) recordTake(account byte, at time.Time) {
	s.Require().NoError(s.store.RecordTake(s.T().Context(), AccountID{15: account}, at))
}

func (s *StoreContractSuite) stats(account byte) Stats {
	stats, err := s.store.Stats(s.T().Context(), AccountID{15: account})
	s.Require().NoError(err)
	return stats
}

func (s *StoreContractSuite) TestAnUnknownAccountHasNoProfileAndNoStats() {
	_, err := s.store.Profile(s.T().Context(), AccountID{15: 1})
	s.Require().ErrorIs(err, ErrNoProfile)

	_, err = s.store.Stats(s.T().Context(), AccountID{15: 1})
	s.Require().ErrorIs(err, ErrNoStats)
}

func (s *StoreContractSuite) TestASavedProfileReadsBackAndASecondReplacesIt() {
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Émile 🌍")))
	profile, err := s.store.Profile(s.T().Context(), AccountID{15: 1})
	s.Require().NoError(err)
	s.Equal(contractProfile(1, "Émile 🌍"), profile)

	renamed := contractProfile(1, "Ada")
	renamed.UpdatedAt = contractAt.Add(time.Hour)
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), renamed))
	profile, err = s.store.Profile(s.T().Context(), AccountID{15: 1})
	s.Require().NoError(err)
	s.Equal(renamed, profile)
}

func (s *StoreContractSuite) TestEachTakeIsCountedByTheDomainsRule() {
	s.recordTake(1, contractAt)
	s.recordTake(1, contractAt.Add(time.Minute))
	s.recordTake(1, contractAt.Add(time.Hour))

	want := Stats{Account: AccountID{15: 1}}.WithTake(contractAt).WithTake(contractAt.Add(time.Minute)).WithTake(contractAt.Add(time.Hour))
	s.Equal(want, s.stats(1))
	s.Equal(uint32(2), s.stats(1).StreakCurrent, "the third take is past UTC midnight")
}

func (s *StoreContractSuite) TestTakesOfOneAccountDoNotCountOnAnother() {
	s.recordTake(1, contractAt)
	s.recordTake(2, contractAt)
	s.recordTake(2, contractAt)

	s.Equal(uint64(1), s.stats(1).TilesTaken)
	s.Equal(uint64(2), s.stats(2).TilesTaken)
}

func (s *StoreContractSuite) TestADeletedAccountLosesBothAndTheOthersKeepTheirs() {
	for account := range byte(2) {
		s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(account+1, "named")))
		s.recordTake(account+1, contractAt)
	}

	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), AccountID{15: 1}))

	_, err := s.store.Profile(s.T().Context(), AccountID{15: 1})
	s.Require().ErrorIs(err, ErrNoProfile)
	_, err = s.store.Stats(s.T().Context(), AccountID{15: 1})
	s.Require().ErrorIs(err, ErrNoStats)
	s.Equal(uint64(1), s.stats(2).TilesTaken)
}

func (s *StoreContractSuite) TestDeletingAnUnknownAccountIsNotAnError() {
	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), AccountID{15: 9}))
}

func (s *StoreContractSuite) TestAnAccountWithStatsOnlyHasNoProfile() {
	s.recordTake(1, contractAt)

	_, err := s.store.Profile(s.T().Context(), AccountID{15: 1})
	s.Require().ErrorIs(err, ErrNoProfile)
}

func (s *StoreContractSuite) TestNamesLeaveOutTheAccountsWithNone() {
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Ada")))
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(2, "Bob")))
	s.recordTake(3, contractAt)

	names, err := s.store.Names(s.T().Context(), []AccountID{{15: 1}, {15: 3}, {15: 4}})

	s.Require().NoError(err)
	s.Equal(map[AccountID]Name{{15: 1}: "Ada"}, names)
}

func (s *StoreContractSuite) TestNoAccountsAskedIsNoNames() {
	names, err := s.store.Names(s.T().Context(), nil)

	s.Require().NoError(err)
	s.Empty(names)
}

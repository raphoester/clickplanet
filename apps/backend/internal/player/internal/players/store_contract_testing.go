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
	// MakeAdmin does what an operator does in the database: the port has no way to.
	MakeAdmin func(store Store, account AccountID)

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
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Emile_1858")))
	profile, err := s.store.Profile(s.T().Context(), AccountID{15: 1})
	s.Require().NoError(err)
	s.Equal(contractProfile(1, "Emile_1858"), profile)

	renamed := contractProfile(1, "Ada")
	renamed.UpdatedAt = contractAt.Add(time.Hour)
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), renamed))
	profile, err = s.store.Profile(s.T().Context(), AccountID{15: 1})
	s.Require().NoError(err)
	s.Equal(renamed, profile)
}

func (s *StoreContractSuite) TestANewProfileIsNotAnAdmin() {
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Ada")))

	profile, err := s.store.Profile(s.T().Context(), AccountID{15: 1})
	s.Require().NoError(err)
	s.False(profile.Admin)
}

func (s *StoreContractSuite) TestAnAdminIsReadAndARenameKeepsIt() {
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Ada")))
	s.MakeAdmin(s.store, AccountID{15: 1})

	notAdmin := contractProfile(1, "Ada_L")
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), notAdmin), "a saved profile says nothing of admin")

	profile, err := s.store.Profile(s.T().Context(), AccountID{15: 1})
	s.Require().NoError(err)
	s.True(profile.Admin)
	s.Equal(Name("Ada_L"), profile.Name)
	named, err := s.store.ProfileNamed(s.T().Context(), "ada_l")
	s.Require().NoError(err)
	s.True(named.Admin)
}

func (s *StoreContractSuite) TestAProfileIsFoundByItsNameIgnoringCase() {
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Ada_L")))
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(2, "Bob")))

	profile, err := s.store.ProfileNamed(s.T().Context(), "aDA_l")

	s.Require().NoError(err)
	s.Equal(contractProfile(1, "Ada_L"), profile)
}

func (s *StoreContractSuite) TestANameNobodyHoldsHasNoProfile() {
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Ada")))
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Bob")))

	_, err := s.store.ProfileNamed(s.T().Context(), "Ada")

	s.Require().ErrorIs(err, ErrNoProfile, "a rename frees the old name")
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
		s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(account+1, []Name{"Ada", "Bob"}[account])))
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

func (s *StoreContractSuite) TestANameAnotherAccountHoldsIsTakenIgnoringCase() {
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Ada_L")))

	err := s.store.SaveProfile(s.T().Context(), contractProfile(2, "aDA_l"))

	s.Require().ErrorIs(err, ErrNameTaken)
	_, err = s.store.Profile(s.T().Context(), AccountID{15: 2})
	s.Require().ErrorIs(err, ErrNoProfile, "a refused name writes nothing")
}

func (s *StoreContractSuite) TestAnAccountSavesItsOwnNameInAnotherCase() {
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Ada_L")))

	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "ADA_L")))

	profile, err := s.store.Profile(s.T().Context(), AccountID{15: 1})
	s.Require().NoError(err)
	s.Equal(Name("ADA_L"), profile.Name)
}

func (s *StoreContractSuite) TestARenameFreesTheOldName() {
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Ada")))
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Bob")))

	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(2, "ada")))
}

func (s *StoreContractSuite) TestAnAccountNeverGivenACodeHasNone() {
	_, err := s.store.GuestCode(s.T().Context(), AccountID{15: 1})

	s.Require().ErrorIs(err, ErrNoGuestCode)
}

func (s *StoreContractSuite) TestASavedGuestCodeReadsBack() {
	s.Require().NoError(s.store.SaveGuestCode(s.T().Context(), AccountID{15: 1}, "a1b2c3"))

	code, err := s.store.GuestCode(s.T().Context(), AccountID{15: 1})

	s.Require().NoError(err)
	s.Equal(GuestCode("a1b2c3"), code)
}

func (s *StoreContractSuite) TestAnAccountKeepsItsFirstGuestCode() {
	s.Require().NoError(s.store.SaveGuestCode(s.T().Context(), AccountID{15: 1}, "a1b2c3"))

	s.Require().NoError(s.store.SaveGuestCode(s.T().Context(), AccountID{15: 1}, "ffffff"))

	code, err := s.store.GuestCode(s.T().Context(), AccountID{15: 1})
	s.Require().NoError(err)
	s.Equal(GuestCode("a1b2c3"), code)
}

func (s *StoreContractSuite) TestAGuestCodeAnotherAccountHoldsIsTaken() {
	s.Require().NoError(s.store.SaveGuestCode(s.T().Context(), AccountID{15: 1}, "a1b2c3"))

	err := s.store.SaveGuestCode(s.T().Context(), AccountID{15: 2}, "a1b2c3")

	s.Require().ErrorIs(err, ErrGuestCodeTaken)
	_, err = s.store.GuestCode(s.T().Context(), AccountID{15: 2})
	s.Require().ErrorIs(err, ErrNoGuestCode, "a refused code writes nothing")
}

func (s *StoreContractSuite) TestADeletedAccountFreesItsGuestCode() {
	s.Require().NoError(s.store.SaveGuestCode(s.T().Context(), AccountID{15: 1}, "a1b2c3"))
	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), AccountID{15: 1}))

	_, err := s.store.GuestCode(s.T().Context(), AccountID{15: 1})
	s.Require().ErrorIs(err, ErrNoGuestCode)
	s.Require().NoError(s.store.SaveGuestCode(s.T().Context(), AccountID{15: 2}, "a1b2c3"))
}

func (s *StoreContractSuite) TestADeletedAccountFreesItsName() {
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Ada")))
	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), AccountID{15: 1}))

	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(2, "Ada")))
}

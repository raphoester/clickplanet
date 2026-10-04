//go:build testing

package players

import (
	"time"

	"github.com/stretchr/testify/suite"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore   func() Store
	MakeAdmin  func(store Store, account AccountID)
	RecordTake func(store Store, account AccountID, at time.Time)

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

var contractAt = time.Date(2026, 9, 17, 23, 30, 0, 123_456_000, time.UTC)

func contractProfile(account byte, name Name) Profile {
	return NewProfile(AccountID{15: account}, name, contractAt)
}

func (s *StoreContractSuite) recordTake(account byte, at time.Time) {
	s.RecordTake(s.store, AccountID{15: account}, at)
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

func (s *StoreContractSuite) TestACreatedProfileReadsBack() {
	s.Require().NoError(s.store.CreateProfile(s.T().Context(), contractProfile(1, "BraveFox42")))

	profile, err := s.store.Profile(s.T().Context(), AccountID{15: 1})
	s.Require().NoError(err)
	s.Equal(contractProfile(1, "BraveFox42"), profile)
}

func (s *StoreContractSuite) TestCreatingAProfileForAnAccountThatHasOneChangesNothing() {
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Ada")))
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(2, "Bob")))

	s.Require().ErrorIs(s.store.CreateProfile(s.T().Context(), contractProfile(1, "BraveFox42")), ErrProfileExists)
	s.Require().ErrorIs(s.store.CreateProfile(s.T().Context(), contractProfile(1, "bob")), ErrProfileExists,
		"an account with a profile is refused before its name is looked at")

	profile, err := s.store.Profile(s.T().Context(), AccountID{15: 1})
	s.Require().NoError(err)
	s.Equal(Name("Ada"), profile.Name())
}

func (s *StoreContractSuite) TestCreatingAProfileWithANameAnotherHoldsIsNameTaken() {
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "BraveFox42")))

	s.Require().ErrorIs(s.store.CreateProfile(s.T().Context(), contractProfile(2, "bravefox42")), ErrNameTaken)

	_, err := s.store.Profile(s.T().Context(), AccountID{15: 2})
	s.Require().ErrorIs(err, ErrNoProfile)
}

func (s *StoreContractSuite) TestASavedProfileReadsBackAndASecondReplacesIt() {
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Emile_1858")))
	profile, err := s.store.Profile(s.T().Context(), AccountID{15: 1})
	s.Require().NoError(err)
	s.Equal(contractProfile(1, "Emile_1858"), profile)

	renamed := NewProfile(AccountID{15: 1}, "Ada", contractAt.Add(time.Hour))
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), renamed))
	profile, err = s.store.Profile(s.T().Context(), AccountID{15: 1})
	s.Require().NoError(err)
	s.Equal(renamed, profile)
}

func (s *StoreContractSuite) TestANewProfileIsNotAnAdmin() {
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Ada")))

	profile, err := s.store.Profile(s.T().Context(), AccountID{15: 1})
	s.Require().NoError(err)
	s.False(profile.Admin())
}

func (s *StoreContractSuite) TestAnAdminIsReadAndARenameKeepsIt() {
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Ada")))
	s.MakeAdmin(s.store, AccountID{15: 1})

	notAdmin := contractProfile(1, "Ada_L")
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), notAdmin), "a saved profile says nothing of admin")

	profile, err := s.store.Profile(s.T().Context(), AccountID{15: 1})
	s.Require().NoError(err)
	s.True(profile.Admin())
	s.Equal(Name("Ada_L"), profile.Name())
}

func (s *StoreContractSuite) TestEachMessageIsCountedAndStartsNoStreak() {
	s.Require().NoError(s.store.RecordMessage(s.T().Context(), AccountID{15: 1}))
	s.Require().NoError(s.store.RecordMessage(s.T().Context(), AccountID{15: 1}))

	s.Equal(NewStats(AccountID{15: 1}).WithMessage().WithMessage(), s.stats(1))
}

func (s *StoreContractSuite) TestAMessageKeepsTheTakesCounted() {
	s.Require().NoError(s.store.RecordMessage(s.T().Context(), AccountID{15: 1}))
	s.recordTake(1, contractAt)
	s.Require().NoError(s.store.RecordMessage(s.T().Context(), AccountID{15: 1}))
	s.Require().NoError(s.store.RecordMessage(s.T().Context(), AccountID{15: 2}))

	s.Equal(NewStats(AccountID{15: 1}).WithMessage().WithTake(contractAt).WithMessage(), s.stats(1))
	s.Equal(uint64(1), s.stats(2).MessagesSent())
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
	s.Equal(uint64(1), s.stats(2).TilesTaken())
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
	s.Equal(Name("ADA_L"), profile.Name())
}

func (s *StoreContractSuite) TestARenameFreesTheOldName() {
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Ada")))
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Bob")))

	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(2, "ada")))
}

func (s *StoreContractSuite) TestStatsArePagedInAccountOrderAfterTheCursor() {
	for _, account := range []byte{3, 1, 4, 2} {
		s.recordTake(account, contractAt)
	}

	first, err := s.store.StatsAfter(s.T().Context(), AccountID{}, 3)
	s.Require().NoError(err)
	rest, err := s.store.StatsAfter(s.T().Context(), first[len(first)-1].Account(), 3)
	s.Require().NoError(err)

	accountsOf := func(page []Stats) []AccountID {
		accounts := make([]AccountID, 0, len(page))
		for _, stats := range page {
			accounts = append(accounts, stats.Account())
		}
		return accounts
	}
	s.Equal([]AccountID{{15: 1}, {15: 2}, {15: 3}}, accountsOf(first))
	s.Equal([]AccountID{{15: 4}}, accountsOf(rest))
	s.Equal(s.stats(4), rest[0])
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

func (s *StoreContractSuite) TestANameOfAnyScriptIsTakenByItsFold() {
	for i, pair := range [][2]Name{{"Émile Zola", "éMILE zOLA"}, {"Straße", "STRASSE"}, {"Жанна", "жАННА"}, {"Ａｄａ", "ada"}} {
		held, asked := contractProfile(byte(2*i+1), pair[0]), contractProfile(byte(2*i+2), pair[1])
		s.Require().NoError(s.store.SaveProfile(s.T().Context(), held))

		s.Require().ErrorIs(s.store.SaveProfile(s.T().Context(), asked), ErrNameTaken, "%q holds %q", pair[0], pair[1])
	}
}

func (s *StoreContractSuite) TestNamesThatOnlyLookAlikeAreTwoNames() {
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Ada")))

	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(2, "Adá")))
}

func (s *StoreContractSuite) TestAColorNeedsAProfile() {
	err := s.store.SaveColor(s.T().Context(), AccountID{15: 1}, 3)

	s.Require().ErrorIs(err, ErrNoProfile, "a guest has no name to color")
}

func (s *StoreContractSuite) TestAColorIsReadAndARenameKeepsIt() {
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Ada")))
	s.Require().NoError(s.store.SaveColor(s.T().Context(), AccountID{15: 1}, 3))
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Ada_L")), "a saved profile says nothing of color")

	profile, err := s.store.Profile(s.T().Context(), AccountID{15: 1})
	s.Require().NoError(err)
	s.Equal(Color(3), profile.Color())
}

func (s *StoreContractSuite) TestAColorCanBeTakenBack() {
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Ada")))
	s.Require().NoError(s.store.SaveColor(s.T().Context(), AccountID{15: 1}, 3))
	s.Require().NoError(s.store.SaveColor(s.T().Context(), AccountID{15: 1}, 0))

	profile, err := s.store.Profile(s.T().Context(), AccountID{15: 1})
	s.Require().NoError(err)
	s.Equal(Color(0), profile.Color())
}

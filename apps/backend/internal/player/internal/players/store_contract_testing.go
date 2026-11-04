//go:build testing

package players

import (
	"time"

	"github.com/stretchr/testify/suite"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore  func() Store
	MakeAdmin func(store Store, account AccountID)

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

var contractAt = time.Date(2026, 9, 17, 23, 30, 0, 123_456_000, time.UTC)

func contractProfile(account byte, name Name) Profile {
	return NewProfile(AccountID{15: account}, name, contractAt)
}

func (s *StoreContractSuite) TestAnUnknownAccountHasNoProfile() {
	_, err := s.store.Profile(s.T().Context(), AccountID{15: 1})
	s.Require().ErrorIs(err, ErrNoProfile)
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
	s.Equal(Name("Ada"), profile.TakeSnapshot().Name)
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

func (s *StoreContractSuite) TestWhenThePlayerChoseItsNameReadsBack() {
	chosen, err := Unnamed(AccountID{15: 1}).Renamed("Ada", contractAt)
	s.Require().NoError(err)
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), chosen))

	profile, err := s.store.Profile(s.T().Context(), AccountID{15: 1})
	s.Require().NoError(err)
	s.Equal(chosen, profile)

	again, err := profile.Renamed("Ada_L", contractAt.Add(renameEvery))
	s.Require().NoError(err)
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), again))
	profile, err = s.store.Profile(s.T().Context(), AccountID{15: 1})
	s.Require().NoError(err)
	s.Equal(again, profile)
}

func (s *StoreContractSuite) TestANewProfileIsNotAnAdmin() {
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Ada")))

	profile, err := s.store.Profile(s.T().Context(), AccountID{15: 1})
	s.Require().NoError(err)
	s.False(profile.TakeSnapshot().Admin)
}

func (s *StoreContractSuite) TestAnAdminIsReadAndARenameKeepsIt() {
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Ada")))
	s.MakeAdmin(s.store, AccountID{15: 1})

	notAdmin := contractProfile(1, "Ada_L")
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), notAdmin), "a saved profile says nothing of admin")

	profile, err := s.store.Profile(s.T().Context(), AccountID{15: 1})
	s.Require().NoError(err)
	s.True(profile.TakeSnapshot().Admin)
	s.Equal(Name("Ada_L"), profile.TakeSnapshot().Name)
}

func (s *StoreContractSuite) TestADeletedAccountLosesItsProfileAndTheOthersKeepTheirs() {
	for account := range byte(2) {
		s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(account+1, []Name{"Ada", "Bob"}[account])))
	}

	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), AccountID{15: 1}))

	_, err := s.store.Profile(s.T().Context(), AccountID{15: 1})
	s.Require().ErrorIs(err, ErrNoProfile)
	_, err = s.store.Profile(s.T().Context(), AccountID{15: 2})
	s.Require().NoError(err)
}

func (s *StoreContractSuite) TestDeletingAnUnknownAccountIsNotAnError() {
	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), AccountID{15: 9}))
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
	s.Equal(Name("ADA_L"), profile.TakeSnapshot().Name)
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
	s.Equal(Color(3), profile.TakeSnapshot().Color)
}

func (s *StoreContractSuite) TestAColorCanBeTakenBack() {
	s.Require().NoError(s.store.SaveProfile(s.T().Context(), contractProfile(1, "Ada")))
	s.Require().NoError(s.store.SaveColor(s.T().Context(), AccountID{15: 1}, 3))
	s.Require().NoError(s.store.SaveColor(s.T().Context(), AccountID{15: 1}, 0))

	profile, err := s.store.Profile(s.T().Context(), AccountID{15: 1})
	s.Require().NoError(err)
	s.Equal(Color(0), profile.TakeSnapshot().Color)
}

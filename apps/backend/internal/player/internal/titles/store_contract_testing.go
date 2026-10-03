//go:build testing

package titles

import (
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

var contractAt = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)

func (s *StoreContractSuite) held(account byte) IDs {
	held, err := s.store.Held(s.T().Context(), players.AccountID{15: account})
	s.Require().NoError(err)
	return held
}

func (s *StoreContractSuite) grant(account byte, ids ...ID) {
	s.Require().NoError(s.store.Grant(s.T().Context(), Holdings{{15: account}: ids}, contractAt))
}

func (s *StoreContractSuite) TestAnAccountNeverGrantedATitleHoldsNone() {
	s.Empty(s.held(1))
}

func (s *StoreContractSuite) TestGrantedTitlesReadBack() {
	s.grant(1, "settler", "loyal")

	s.ElementsMatch(IDs{"settler", "loyal"}, s.held(1))
}

func (s *StoreContractSuite) TestATitleGrantedAgainIsHeldOnce() {
	s.grant(1, "settler")

	s.grant(1, "settler", "governor")

	s.ElementsMatch(IDs{"settler", "governor"}, s.held(1))
}

func (s *StoreContractSuite) TestOneGrantGivesManyAccountsTheirOwnTitles() {
	s.Require().NoError(s.store.Grant(s.T().Context(), Holdings{
		{15: 1}: {"settler"},
		{15: 2}: {"loyal", "devoted"},
	}, contractAt))

	s.Equal(IDs{"settler"}, s.held(1))
	s.ElementsMatch(IDs{"loyal", "devoted"}, s.held(2))
	s.Empty(s.held(3))
}

func (s *StoreContractSuite) TestAnEmptyGrantIsNotAnError() {
	s.Require().NoError(s.store.Grant(s.T().Context(), Holdings{}, contractAt))
}

func (s *StoreContractSuite) TestADeletedAccountLosesItsTitlesAndTheOthersKeepTheirs() {
	s.grant(1, "settler")
	s.grant(2, "settler")

	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), players.AccountID{15: 1}))

	s.Empty(s.held(1))
	s.Equal(IDs{"settler"}, s.held(2))
}

func (s *StoreContractSuite) TestDeletingAnAccountWithNoTitleIsNotAnError() {
	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), players.AccountID{15: 9}))
}

func (s *StoreContractSuite) TestHoldingsAnswerThePageAndLeaveOutAccountsWithNoTitle() {
	s.grant(1, "settler", "loyal")
	s.grant(2, "og")
	s.grant(3, "og")

	holdings, err := s.store.Holdings(s.T().Context(), []players.AccountID{{15: 1}, {15: 2}, {15: 4}})

	s.Require().NoError(err)
	s.Require().Len(holdings, 2)
	s.ElementsMatch(IDs{"settler", "loyal"}, holdings[players.AccountID{15: 1}])
	s.Equal(IDs{"og"}, holdings[players.AccountID{15: 2}])
}

func (s *StoreContractSuite) TestNoAccountsAskedIsNoHoldings() {
	holdings, err := s.store.Holdings(s.T().Context(), nil)

	s.Require().NoError(err)
	s.Empty(holdings)
}

func (s *StoreContractSuite) TestARevokedTitleIsNoLongerHeldAndTheRestStay() {
	s.grant(1, "settler", "governor")
	s.grant(2, "governor")

	s.Require().NoError(s.store.Revoke(s.T().Context(), Holdings{{15: 1}: {"governor"}}))

	s.Equal(IDs{"settler"}, s.held(1))
	s.Equal(IDs{"governor"}, s.held(2))
}

func (s *StoreContractSuite) TestRevokingATitleNotHeldOrNothingIsNotAnError() {
	s.grant(1, "settler")

	s.Require().NoError(s.store.Revoke(s.T().Context(), Holdings{{15: 1}: {"emperor"}, {15: 2}: {"og"}}))
	s.Require().NoError(s.store.Revoke(s.T().Context(), Holdings{}))

	s.Equal(IDs{"settler"}, s.held(1))
}

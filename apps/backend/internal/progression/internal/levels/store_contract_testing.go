//go:build testing

package levels

import (
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

var contractAt = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

func (s *StoreContractSuite) levels(asked ...byte) ByAccount {
	ids := make([]accounts.AccountID, 0, len(asked))
	for _, account := range asked {
		ids = append(ids, accounts.AccountID{15: account})
	}
	kept, err := s.store.Levels(s.T().Context(), ids)
	s.Require().NoError(err)
	return kept
}

func (s *StoreContractSuite) raise(account byte, level Level) Level {
	previous, err := s.store.Raise(s.T().Context(), accounts.AccountID{15: account}, level, contractAt)
	s.Require().NoError(err)
	return previous
}

func (s *StoreContractSuite) TestAnAccountNeverRaisedHasNoLevel() {
	s.Empty(s.levels(1))
}

func (s *StoreContractSuite) TestNoAccountAskedIsNoLevels() {
	s.Empty(s.levels())
}

func (s *StoreContractSuite) TestARaiseKeepsTheLevelAndAnswersTheOneBefore() {
	s.Equal(Level(0), s.raise(1, 3), "an account with no level")
	s.Equal(Level(3), s.raise(1, 9), "one jump, however many levels")

	s.Equal(ByAccount{{15: 1}: 9}, s.levels(1, 2))
}

func (s *StoreContractSuite) TestARaiseNeverLowersAndAnswersTheLevelKept() {
	s.raise(1, 9)

	s.Equal(Level(9), s.raise(1, 4))
	s.Equal(Level(9), s.raise(1, 9), "the same level is no raise")
	s.Equal(ByAccount{{15: 1}: 9}, s.levels(1))
}

func (s *StoreContractSuite) TestRaiseAllRaisesEachAccountAndLowersNone() {
	s.raise(1, 9)
	s.raise(2, 2)

	s.Require().NoError(s.store.RaiseAll(s.T().Context(), ByAccount{{15: 1}: 4, {15: 2}: 5, {15: 3}: 1}, contractAt))

	s.Equal(ByAccount{{15: 1}: 9, {15: 2}: 5, {15: 3}: 1}, s.levels(1, 2, 3))
}

func (s *StoreContractSuite) TestSetKeepsEachLevelAsItIsLowerOrHigher() {
	s.raise(1, 9)
	s.raise(2, 2)

	s.Require().NoError(s.store.Set(s.T().Context(), ByAccount{{15: 1}: 4, {15: 2}: 5, {15: 3}: 1}, contractAt))

	s.Equal(ByAccount{{15: 1}: 4, {15: 2}: 5, {15: 3}: 1}, s.levels(1, 2, 3))
	s.Equal(Level(4), s.raise(1, 6), "a raise reads the level set")
}

func (s *StoreContractSuite) TestNothingToRaiseOrSetIsNotAnError() {
	s.Require().NoError(s.store.RaiseAll(s.T().Context(), ByAccount{}, contractAt))
	s.Require().NoError(s.store.Set(s.T().Context(), ByAccount{}, contractAt))
}

func (s *StoreContractSuite) TestAForgottenAccountHasNoLevelAndTheOthersKeepTheirs() {
	s.raise(1, 9)
	s.raise(2, 2)
	s.raise(3, 5)

	s.Require().NoError(s.store.Forget(s.T().Context(), accounts.AccountID{15: 1}, accounts.AccountID{15: 3}))

	s.Equal(ByAccount{{15: 2}: 2}, s.levels(1, 2, 3))
	s.Equal(Level(0), s.raise(1, 2), "raised again, it starts from none")
}

func (s *StoreContractSuite) TestForgettingNoneOrAnAccountWithNoLevelIsNotAnError() {
	s.Require().NoError(s.store.Forget(s.T().Context()))
	s.Require().NoError(s.store.Forget(s.T().Context(), accounts.AccountID{15: 9}))
}

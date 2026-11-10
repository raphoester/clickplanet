//go:build testing

package players

import (
	"context"

	"github.com/stretchr/testify/suite"
)

type ReadableStore interface {
	Store
	IsPlayer(ctx context.Context, account AccountID) (bool, error)
}

type StoreContractSuite struct {
	suite.Suite

	NewStore func() ReadableStore

	store ReadableStore
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

func (s *StoreContractSuite) isPlayer(account AccountID) bool {
	is, err := s.store.IsPlayer(s.T().Context(), account)
	s.Require().NoError(err)
	return is
}

func (s *StoreContractSuite) TestAnAccountPlayerNeverToldOfIsAGuest() {
	s.False(s.isPlayer(AccountID{15: 1}))
}

func (s *StoreContractSuite) TestAKeptPlayerIsOneAndKeptTwiceIsOne() {
	s.Require().NoError(s.store.Keep(s.T().Context(), AccountID{15: 1}))
	s.Require().NoError(s.store.Keep(s.T().Context(), AccountID{15: 1}))

	s.True(s.isPlayer(AccountID{15: 1}))
	s.False(s.isPlayer(AccountID{15: 2}))
}

func (s *StoreContractSuite) TestAForgottenPlayerIsNoLongerOne() {
	s.Require().NoError(s.store.Keep(s.T().Context(), AccountID{15: 1}))
	s.Require().NoError(s.store.Keep(s.T().Context(), AccountID{15: 2}))

	s.Require().NoError(s.store.Forget(s.T().Context(), AccountID{15: 1}))
	s.Require().NoError(s.store.Forget(s.T().Context(), AccountID{15: 9}))

	s.False(s.isPlayer(AccountID{15: 1}))
	s.True(s.isPlayer(AccountID{15: 2}))
}

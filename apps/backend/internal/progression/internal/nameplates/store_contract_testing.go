//go:build testing

package nameplates

import (
	"context"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
)

type ReadableStore interface {
	Store
	Named(ctx context.Context, account accounts.AccountID) (bool, error)
}

type StoreContractSuite struct {
	suite.Suite

	NewStore func() ReadableStore

	store ReadableStore
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

func (s *StoreContractSuite) named(account accounts.AccountID) bool {
	named, err := s.store.Named(s.T().Context(), account)
	s.Require().NoError(err)
	return named
}

func (s *StoreContractSuite) TestAnAccountPlayerNeverNamedIsNotNamed() {
	s.False(s.named(accounts.AccountID{15: 1}))
}

func (s *StoreContractSuite) TestAKeptNameplateIsNamedAndKeptTwiceIsOne() {
	s.Require().NoError(s.store.Keep(s.T().Context(), accounts.AccountID{15: 1}))
	s.Require().NoError(s.store.Keep(s.T().Context(), accounts.AccountID{15: 1}))

	s.True(s.named(accounts.AccountID{15: 1}))
	s.False(s.named(accounts.AccountID{15: 2}))
}

func (s *StoreContractSuite) TestAForgottenNameplateIsNoLongerNamed() {
	s.Require().NoError(s.store.Keep(s.T().Context(), accounts.AccountID{15: 1}))
	s.Require().NoError(s.store.Keep(s.T().Context(), accounts.AccountID{15: 2}))

	s.Require().NoError(s.store.Forget(s.T().Context(), accounts.AccountID{15: 1}))
	s.Require().NoError(s.store.Forget(s.T().Context(), accounts.AccountID{15: 9}))

	s.False(s.named(accounts.AccountID{15: 1}))
	s.True(s.named(accounts.AccountID{15: 2}))
}

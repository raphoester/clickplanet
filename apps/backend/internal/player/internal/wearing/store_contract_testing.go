//go:build testing

package wearing

import (
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

var contractAt = time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)

func (s *StoreContractSuite) choice(account byte) titles.ID {
	choice, err := s.store.Choice(s.T().Context(), players.AccountID{15: account})
	s.Require().NoError(err)
	return choice
}

func (s *StoreContractSuite) wear(account byte, title titles.ID, at time.Time) {
	s.Require().NoError(s.store.Wear(s.T().Context(), players.AccountID{15: account}, title, at))
}

func (s *StoreContractSuite) TestAnAccountThatNeverChoseHasNoChoice() {
	s.Empty(s.choice(1))
}

func (s *StoreContractSuite) TestTheLastTitleWornIsTheChoice() {
	s.wear(1, "settler", contractAt)
	s.wear(1, "og", contractAt.Add(time.Hour))
	s.wear(2, "loyal", contractAt)

	s.Equal(titles.ID("og"), s.choice(1))
	s.Equal(titles.ID("loyal"), s.choice(2))
}

func (s *StoreContractSuite) TestADeletedAccountHasNoChoiceAndTheOthersKeepTheirs() {
	s.wear(1, "settler", contractAt)
	s.wear(2, "og", contractAt)

	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), players.AccountID{15: 1}))

	s.Empty(s.choice(1))
	s.Equal(titles.ID("og"), s.choice(2))
}

func (s *StoreContractSuite) TestDeletingAnAccountThatNeverChoseIsNotAnError() {
	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), players.AccountID{15: 9}))
}

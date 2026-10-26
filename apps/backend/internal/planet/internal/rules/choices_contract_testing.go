//go:build testing

package rules

import (
	"time"

	"github.com/stretchr/testify/suite"
)

type ChoiceStoreContractSuite struct {
	suite.Suite

	NewStore func() Choices

	store Choices
}

func (s *ChoiceStoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

var chosenAt = time.Date(2026, 10, 31, 21, 0, 0, 123456000, time.UTC)

func (s *ChoiceStoreContractSuite) TestNoChoiceIsKeptAtFirst() {
	_, kept, err := s.store.Choice(s.T().Context())
	s.Require().NoError(err)
	s.False(kept)
}

func (s *ChoiceStoreContractSuite) TestAChoiceIsReadAsItWasKept() {
	s.Require().NoError(s.store.Save(s.T().Context(), Choice{Set: "rush", DecidedAt: chosenAt}))

	choice, kept, err := s.store.Choice(s.T().Context())
	s.Require().NoError(err)
	s.True(kept)
	s.Equal(SetName("rush"), choice.Set)
	s.True(chosenAt.Equal(choice.DecidedAt), "%s", choice.DecidedAt)
}

func (s *ChoiceStoreContractSuite) TestAChoiceKeptReplacesTheOneBefore() {
	s.Require().NoError(s.store.Save(s.T().Context(), Choice{Set: "rush", DecidedAt: chosenAt}))
	s.Require().NoError(s.store.Save(s.T().Context(), Choice{Set: Default, DecidedAt: chosenAt.Add(2 * time.Hour)}))

	choice, _, err := s.store.Choice(s.T().Context())
	s.Require().NoError(err)
	s.Equal(Default, choice.Set)
	s.True(chosenAt.Add(2 * time.Hour).Equal(choice.DecidedAt))
}

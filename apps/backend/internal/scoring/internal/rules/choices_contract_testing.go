//go:build testing

package rules

import (
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds"
)

type ChoiceStoreContractSuite struct {
	suite.Suite

	NewStore func() Choices

	store Choices
}

func (s *ChoiceStoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

var (
	chosenAt = time.Date(2026, 10, 31, 21, 0, 0, 123456000, time.UTC)
	battle   = Window{
		Entry: rounds.EntryID(uuid.MustParse("0192f5a0-7c00-7000-8000-000000000031")), StartsAt: chosenAt,
		EndsAt: chosenAt.Add(2 * time.Hour),
	}
)

func (s *ChoiceStoreContractSuite) TestNoChoiceIsKeptAtFirst() {
	_, kept, err := s.store.Choice(s.T().Context())
	s.Require().NoError(err)
	s.False(kept)
}

func (s *ChoiceStoreContractSuite) TestAChoiceIsReadAsItWasKept() {
	s.Require().NoError(s.store.Save(s.T().Context(), Choice{Set: "rush", Window: battle, DecidedAt: chosenAt}))

	choice, kept, err := s.store.Choice(s.T().Context())
	s.Require().NoError(err)
	s.True(kept)
	s.Equal(SetName("rush"), choice.Set)
	s.True(chosenAt.Equal(choice.DecidedAt), "%s", choice.DecidedAt)
	s.True(battle.StartsAt.Equal(choice.Window.StartsAt), "%s", choice.Window.StartsAt)
	s.True(battle.EndsAt.Equal(choice.Window.EndsAt), "%s", choice.Window.EndsAt)
	s.Equal(battle.Entry, choice.Window.Entry)
}

func (s *ChoiceStoreContractSuite) TestAChoiceKeptReplacesTheOneBefore() {
	s.Require().NoError(s.store.Save(s.T().Context(), Choice{Set: "rush", Window: battle, DecidedAt: chosenAt}))
	s.Require().NoError(s.store.Save(s.T().Context(), Choice{Set: Default, DecidedAt: chosenAt.Add(2 * time.Hour)}))

	choice, _, err := s.store.Choice(s.T().Context())
	s.Require().NoError(err)
	s.Equal(Default, choice.Set)
	s.True(choice.Window.None())
	s.True(chosenAt.Add(2 * time.Hour).Equal(choice.DecidedAt))
}

//go:build testing

package moments

import (
	"time"

	"github.com/stretchr/testify/suite"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

var contractEnd = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func (s *StoreContractSuite) TestARoundIsKeptOnce() {
	scored := ScoredRound{Round: "season-0/day-24", Book: "season-0", EndedAt: contractEnd}

	first, err := s.store.KeepRound(s.T().Context(), scored)
	s.Require().NoError(err)
	again, err := s.store.KeepRound(s.T().Context(), scored)
	s.Require().NoError(err)

	s.True(first)
	s.False(again, "a round told twice is kept once")
}

func (s *StoreContractSuite) TestABookIsKeptOnce() {
	closed := ClosedBook{Book: "season-0", EndedAt: contractEnd.Add(2 * time.Hour)}

	first, err := s.store.KeepBook(s.T().Context(), closed)
	s.Require().NoError(err)
	again, err := s.store.KeepBook(s.T().Context(), closed)
	s.Require().NoError(err)

	s.True(first)
	s.False(again)
}

func (s *StoreContractSuite) TestTheRoundsOpenedAreReadAsKeptOnce() {
	s.Require().NoError(s.store.KeepOpened(s.T().Context(), "season-0/day-24"))
	s.Require().NoError(s.store.KeepOpened(s.T().Context(), "season-0/day-24"))
	s.Require().NoError(s.store.KeepOpened(s.T().Context(), "season-0/day-25"))

	opened, err := s.store.Opened(s.T().Context())
	s.Require().NoError(err)
	s.ElementsMatch([]RoundID{"season-0/day-24", "season-0/day-25"}, opened)
}

func (s *StoreContractSuite) TestARoundAndABookOfOneSeasonAreEachKept() {
	kept, err := s.store.KeepRound(s.T().Context(), ScoredRound{Round: "season-0/day-1", Book: "season-0", EndedAt: contractEnd})
	s.Require().NoError(err)
	s.True(kept)
	kept, err = s.store.KeepBook(s.T().Context(), ClosedBook{Book: "season-0", EndedAt: contractEnd})
	s.Require().NoError(err)
	s.True(kept)
}

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

func (s *StoreContractSuite) TestTheRoundsScoredAreReadAsKept() {
	ctx := s.T().Context()
	at := time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)
	for _, round := range []RoundID{"season-0/day-24", "season-0/day-24", "season-0/finale"} {
		_, err := s.store.KeepRound(ctx, ScoredRound{Round: round, Book: "season-0", EndedAt: at})
		s.Require().NoError(err)
	}

	scored, err := s.store.Scored(ctx)

	s.Require().NoError(err)
	s.ElementsMatch([]RoundID{"season-0/day-24", "season-0/finale"}, scored)
}

func (s *StoreContractSuite) TestASeasonIsToldOnce() {
	first, err := s.store.KeepToldSeason(s.T().Context(), "season-1")
	s.Require().NoError(err)
	again, err := s.store.KeepToldSeason(s.T().Context(), "season-1")
	s.Require().NoError(err)
	next, err := s.store.KeepToldSeason(s.T().Context(), "season-2")
	s.Require().NoError(err)

	s.True(first)
	s.False(again)
	s.True(next)
}

func (s *StoreContractSuite) TestADayIsToldOnceAndAgainWhenItsEndMoves() {
	day := ToldDay{Round: "season-0/day-24", EndsAt: contractEnd}
	cut := ToldDay{Round: "season-0/day-24", EndsAt: contractEnd.Add(-time.Hour)}

	first, err := s.store.KeepToldDay(s.T().Context(), day)
	s.Require().NoError(err)
	again, err := s.store.KeepToldDay(s.T().Context(), day)
	s.Require().NoError(err)
	moved, err := s.store.KeepToldDay(s.T().Context(), cut)
	s.Require().NoError(err)

	s.True(first)
	s.False(again)
	s.True(moved, "a moved season end cut the day")
}

func (s *StoreContractSuite) TestABattleIsToldOnce() {
	first, err := s.store.KeepToldBattle(s.T().Context(), "season-0/finale")
	s.Require().NoError(err)
	again, err := s.store.KeepToldBattle(s.T().Context(), "season-0/finale")
	s.Require().NoError(err)
	next, err := s.store.KeepToldBattle(s.T().Context(), "season-1/finale")
	s.Require().NoError(err)

	s.True(first)
	s.False(again)
	s.True(next)
}

func (s *StoreContractSuite) TestARoundAndABookOfOneSeasonAreEachKept() {
	kept, err := s.store.KeepRound(s.T().Context(), ScoredRound{Round: "season-0/day-1", Book: "season-0", EndedAt: contractEnd})
	s.Require().NoError(err)
	s.True(kept)
	kept, err = s.store.KeepBook(s.T().Context(), ClosedBook{Book: "season-0", EndedAt: contractEnd})
	s.Require().NoError(err)
	s.True(kept)
}

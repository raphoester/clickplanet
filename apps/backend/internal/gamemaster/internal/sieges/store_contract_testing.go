//go:build testing

package sieges

import (
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

const contractBattle moments.RoundID = "0192f5a0-7c00-7000-8000-000000000031"

func (s *StoreContractSuite) TestABattleIsJudgedOnceAndKeepsItsMode() {
	ctx := s.T().Context()

	first, err := s.store.KeepJudged(ctx, moments.Judgement{Round: contractBattle, Mode: moments.Siege})
	s.Require().NoError(err)
	again, err := s.store.KeepJudged(ctx, moments.Judgement{Round: contractBattle, Mode: moments.Plain})
	s.Require().NoError(err)
	_, err = s.store.KeepJudged(ctx, moments.Judgement{Round: "season-1/finale", Mode: moments.Plain})
	s.Require().NoError(err)

	s.True(first)
	s.False(again)
	judged, err := s.store.Judged(ctx)
	s.Require().NoError(err)
	s.ElementsMatch([]moments.Judgement{
		{Round: contractBattle, Mode: moments.Siege}, {Round: "season-1/finale", Mode: moments.Plain},
	}, judged)
}

func (s *StoreContractSuite) TestAnArenaIsKeptClosedOnce() {
	ctx := s.T().Context()

	first, err := s.store.KeepClosed(ctx, contractBattle)
	s.Require().NoError(err)
	again, err := s.store.KeepClosed(ctx, contractBattle)
	s.Require().NoError(err)

	s.True(first)
	s.False(again)
	closed, err := s.store.Closed(ctx)
	s.Require().NoError(err)
	s.Equal([]moments.RoundID{contractBattle}, closed)
}

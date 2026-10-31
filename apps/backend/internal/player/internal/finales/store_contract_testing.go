//go:build testing

package finales

import (
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

func (s *StoreContractSuite) record(snapshots ...FightSnapshot) {
	fights := make([]Fight, 0, len(snapshots))
	for _, snapshot := range snapshots {
		fight, err := snapshot.Restore()
		s.Require().NoError(err)
		fights = append(fights, fight)
	}
	s.Require().NoError(s.store.Record(s.T().Context(), fights))
}

func (s *StoreContractSuite) fights(accounts ...players.AccountID) map[players.AccountID][]FightSnapshot {
	found, err := s.store.Fights(s.T().Context(), accounts)
	s.Require().NoError(err)

	snapshots := map[players.AccountID][]FightSnapshot{}
	for account, fights := range found {
		for _, fight := range fights {
			snapshots[account] = append(snapshots[account], fight.TakeSnapshot())
		}
	}
	return snapshots
}

var (
	ada = players.AccountID{15: 1}
	bob = players.AccountID{15: 2}
)

func (s *StoreContractSuite) TestRecordedFightsReadBackByAccountInSeasonOrder() {
	s.record(
		FightSnapshot{Account: ada, Season: 1, Country: "de", Rank: 4},
		FightSnapshot{Account: ada, Season: 0, Country: "fr", Rank: 1},
		FightSnapshot{Account: bob, Season: 0, Country: "tv", Rank: 0},
	)

	s.Equal(map[players.AccountID][]FightSnapshot{
		ada: {{Account: ada, Season: 0, Country: "fr", Rank: 1}, {Account: ada, Season: 1, Country: "de", Rank: 4}},
		bob: {{Account: bob, Season: 0, Country: "tv", Rank: 0}},
	}, s.fights(ada, bob))
}

func (s *StoreContractSuite) TestAFightRecordedAgainKeepsTheFirst() {
	s.record(FightSnapshot{Account: ada, Season: 0, Country: "fr", Rank: 1})

	s.record(FightSnapshot{Account: ada, Season: 0, Country: "de", Rank: 2})

	s.Equal(map[players.AccountID][]FightSnapshot{
		ada: {{Account: ada, Season: 0, Country: "fr", Rank: 1}},
	}, s.fights(ada))
}

func (s *StoreContractSuite) TestAnAccountThatFoughtNoFinaleIsLeftOut() {
	s.record(FightSnapshot{Account: ada, Season: 0, Country: "fr", Rank: 1})

	s.Equal(map[players.AccountID][]FightSnapshot{
		ada: {{Account: ada, Season: 0, Country: "fr", Rank: 1}},
	}, s.fights(ada, bob))
}

func (s *StoreContractSuite) TestNoAccountsAskedAndNoFightsRecordedAreNotErrors() {
	s.Require().NoError(s.store.Record(s.T().Context(), nil))

	s.Empty(s.fights())
}

func (s *StoreContractSuite) TestADeletedAccountLosesItsFightsAndTheOthersKeepTheirs() {
	s.record(
		FightSnapshot{Account: ada, Season: 0, Country: "fr", Rank: 1},
		FightSnapshot{Account: bob, Season: 0, Country: "fr", Rank: 1},
	)

	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), ada))
	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), players.AccountID{15: 9}))

	s.Equal(map[players.AccountID][]FightSnapshot{
		bob: {{Account: bob, Season: 0, Country: "fr", Rank: 1}},
	}, s.fights(ada, bob))
}

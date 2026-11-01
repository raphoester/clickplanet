//go:build testing

package podiums

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

func (s *StoreContractSuite) record(snapshots ...PlaceSnapshot) {
	places := make([]Place, 0, len(snapshots))
	for _, snapshot := range snapshots {
		place, err := snapshot.Restore()
		s.Require().NoError(err)
		places = append(places, place)
	}
	s.Require().NoError(s.store.Record(s.T().Context(), places))
}

func (s *StoreContractSuite) places(accounts ...players.AccountID) map[players.AccountID][]PlaceSnapshot {
	found, err := s.store.Places(s.T().Context(), accounts)
	s.Require().NoError(err)

	snapshots := map[players.AccountID][]PlaceSnapshot{}
	for account, places := range found {
		for _, place := range places {
			snapshots[account] = append(snapshots[account], place.TakeSnapshot())
		}
	}
	return snapshots
}

var (
	ada = players.AccountID{15: 1}
	bob = players.AccountID{15: 2}
)

func (s *StoreContractSuite) TestRecordedPlacesReadBackByAccountInSeasonOrder() {
	s.record(
		PlaceSnapshot{Account: ada, Season: 1, Rank: 3},
		PlaceSnapshot{Account: ada, Season: 0, Rank: 1},
		PlaceSnapshot{Account: bob, Season: 0, Rank: 2},
	)

	s.Equal(map[players.AccountID][]PlaceSnapshot{
		ada: {{Account: ada, Season: 0, Rank: 1}, {Account: ada, Season: 1, Rank: 3}},
		bob: {{Account: bob, Season: 0, Rank: 2}},
	}, s.places(ada, bob))
}

func (s *StoreContractSuite) TestAPlaceRecordedAgainTakesTheNewRank() {
	s.record(PlaceSnapshot{Account: ada, Season: 0, Rank: 2})

	s.record(PlaceSnapshot{Account: ada, Season: 0, Rank: 1})

	s.Equal(map[players.AccountID][]PlaceSnapshot{ada: {{Account: ada, Season: 0, Rank: 1}}}, s.places(ada))
}

func (s *StoreContractSuite) TestAnAccountOffEveryPodiumIsLeftOut() {
	s.record(PlaceSnapshot{Account: ada, Season: 0, Rank: 1})

	s.Equal(map[players.AccountID][]PlaceSnapshot{ada: {{Account: ada, Season: 0, Rank: 1}}}, s.places(ada, bob))
}

func (s *StoreContractSuite) TestNoAccountsAskedAndNoPlacesRecordedAreNotErrors() {
	s.Require().NoError(s.store.Record(s.T().Context(), nil))

	s.Empty(s.places())
}

func (s *StoreContractSuite) TestADeletedAccountLosesItsPlacesAndTheOthersKeepTheirs() {
	s.record(PlaceSnapshot{Account: ada, Season: 0, Rank: 1}, PlaceSnapshot{Account: bob, Season: 0, Rank: 2})

	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), ada))
	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), players.AccountID{15: 9}))

	s.Equal(map[players.AccountID][]PlaceSnapshot{bob: {{Account: bob, Season: 0, Rank: 2}}}, s.places(ada, bob))
}

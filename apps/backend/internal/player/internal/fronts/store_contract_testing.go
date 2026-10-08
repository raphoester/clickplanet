//go:build testing

package fronts

import (
	"sync"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store
	TallyOf  func(store Store, account players.AccountID) Tally

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

func (s *StoreContractSuite) take(account byte, country, previous Country, tiles int) {
	take, err := NewTake(players.AccountID{15: account}, country, previous)
	s.Require().NoError(err)
	for range tiles {
		s.Require().NoError(s.store.RecordTake(s.T().Context(), take))
	}
}

func (s *StoreContractSuite) tally(account byte) Tally {
	return s.TallyOf(s.store, players.AccountID{15: account})
}

func (s *StoreContractSuite) TestAnAccountThatTookNothingHasAnEmptyTally() {
	s.Equal(TallyOf(nil, nil), s.tally(1))
}

func (s *StoreContractSuite) TestEachTakeCountsForItsFlagAndAgainstTheTilesOwner() {
	s.take(1, "fr", "de", 3)
	s.take(1, "fr", "", 2)
	s.take(1, "it", "fr", 1)

	s.Equal(TallyOf(map[Country]uint64{"fr": 5, "it": 1}, map[Country]uint64{"de": 3, "fr": 1}), s.tally(1))
}

func (s *StoreContractSuite) TestADeletedAccountHasNoTallyAndTheOthersKeepTheirs() {
	s.take(1, "fr", "de", 2)
	s.take(2, "de", "fr", 1)

	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), players.AccountID{15: 1}))

	s.Equal(TallyOf(nil, nil), s.tally(1))
	s.Equal(TallyOf(map[Country]uint64{"de": 1}, map[Country]uint64{"fr": 1}), s.tally(2))
}

func (s *StoreContractSuite) TestDeletingAnAccountThatTookNothingIsNotAnError() {
	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), players.AccountID{15: 9}))
}

func (s *StoreContractSuite) TestTakesAtOnceAreAllCounted() {
	frenchTake, err := NewTake(players.AccountID{15: 1}, "fr", "de")
	s.Require().NoError(err)
	germanTake, err := NewTake(players.AccountID{15: 1}, "de", "fr")
	s.Require().NoError(err)

	var wg sync.WaitGroup
	for i := range 40 {
		take := frenchTake
		if i%2 == 1 {
			take = germanTake
		}
		wg.Go(func() {
			s.NoError(s.store.RecordTake(s.T().Context(), take))
		})
	}
	wg.Wait()

	s.Equal(TallyOf(map[Country]uint64{"fr": 20, "de": 20}, map[Country]uint64{"fr": 20, "de": 20}), s.tally(1))
}

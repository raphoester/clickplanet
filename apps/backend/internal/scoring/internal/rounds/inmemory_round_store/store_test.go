package inmemory_round_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/inmemory_round_store"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &rounds.StoreContractSuite{NewStore: func() rounds.Store { return inmemory_round_store.New() }})
}

func TestTheMemoryStoreCountsTheTakesAsTheContractSays(t *testing.T) {
	var store *inmemory_round_store.Store
	suite.Run(t, &rounds.TakesContractSuite{
		NewStores: func() (rounds.Store, rounds.Takes) {
			store = inmemory_round_store.New()
			return store, store
		},
		CountedIn: func(round rounds.Round) rounds.Counted { return store.CountedIn(round) },
	})
}

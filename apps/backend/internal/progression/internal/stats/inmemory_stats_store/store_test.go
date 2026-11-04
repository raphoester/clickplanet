package inmemory_stats_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/inmemory_stats_store"
)

func TestInMemoryStoreHoldsTheContract(t *testing.T) {
	s := new(stats.StoreContractSuite)
	s.NewStore = func() stats.Store { return inmemory_stats_store.New() }
	suite.Run(t, s)
}

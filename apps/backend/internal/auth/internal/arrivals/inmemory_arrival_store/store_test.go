package inmemory_arrival_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals/inmemory_arrival_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, &arrivals.StoreContractSuite{
		NewStore: func() arrivals.Store { return inmemory_arrival_store.New() },
		Kept:     func(store arrivals.Store) []arrivals.Snapshot { return store.(*inmemory_arrival_store.Store).Kept() },
	})
}

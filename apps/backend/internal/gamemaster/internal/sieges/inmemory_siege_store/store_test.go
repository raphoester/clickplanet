package inmemory_siege_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/sieges"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/sieges/inmemory_siege_store"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &sieges.StoreContractSuite{NewStore: func() sieges.Store { return inmemory_siege_store.New() }})
}

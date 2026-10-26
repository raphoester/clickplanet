package inmemory_entry_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule/inmemory_entry_store"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &schedule.StoreContractSuite{NewStore: func() schedule.Store { return inmemory_entry_store.New() }})
}

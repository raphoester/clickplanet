package inmemory_arena_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/inmemory_arena_store"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &arenas.StoreContractSuite{NewStore: func() arenas.Store { return inmemory_arena_store.New() }})
}

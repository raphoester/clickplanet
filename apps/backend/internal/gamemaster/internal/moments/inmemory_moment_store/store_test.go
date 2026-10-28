package inmemory_moment_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/inmemory_moment_store"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &moments.StoreContractSuite{NewStore: func() moments.Store { return inmemory_moment_store.New() }})
}

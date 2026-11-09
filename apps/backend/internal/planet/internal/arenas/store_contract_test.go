package arenas_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &arenas.StoreContractSuite{NewStore: func() arenas.Store { return arenas.NewMemoryStore() }})
}

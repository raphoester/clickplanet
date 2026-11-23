package catalog_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/catalog"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &catalog.StoreContractSuite{
		NewStore: func() catalog.Store { return catalog.NewMemoryStore() },
	})
}

package rallies_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rallies"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &rallies.StoreContractSuite{NewStore: func() rallies.Store { return rallies.NewMemoryStore() }})
}

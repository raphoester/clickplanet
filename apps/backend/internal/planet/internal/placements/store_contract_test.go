package placements_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/placements"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &placements.StoreContractSuite{NewStore: func() placements.Store { return placements.NewMemoryStore() }})
}

package rules_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

func TestTheMemoryLayerStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &rules.LayerStoreContractSuite{
		NewStore: func() rules.LayerStore { return rules.NewMemoryLayerStore() },
	})
}

package rules_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/parts"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &rules.StoreContractSuite{
		NewStore: func() rules.Store { return rules.NewMemoryStore(parts.All()) },
		Parts:    parts.All(),
	})
}

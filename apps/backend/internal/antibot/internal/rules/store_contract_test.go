package rules_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &rules.StoreContractSuite{NewStore: func() rules.Store { return rules.NewMemoryStore() }})
}

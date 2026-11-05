package rules_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &rules.StoreContractSuite{
		Parts:    guard.Parts(),
		NewStore: func() rules.Store { return rules.NewMemoryStore(guard.Parts()) },
	})
}

func TestTheMemoryChoicesKeepTheContract(t *testing.T) {
	suite.Run(t, &rules.ChoiceStoreContractSuite{
		NewStore: func() rules.Choices { return rules.NewMemoryChoices() },
	})
}

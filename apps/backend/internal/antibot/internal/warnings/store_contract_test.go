package warnings_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/warnings"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &warnings.StoreContractSuite{NewStore: func() warnings.Store { return warnings.NewMemoryStore() }})
}

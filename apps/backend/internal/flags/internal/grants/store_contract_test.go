package grants_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &grants.StoreContractSuite{
		NewStore: func() grants.Store { return grants.NewMemoryStore() },
		Flag:     "r-0000000001",
	})
}

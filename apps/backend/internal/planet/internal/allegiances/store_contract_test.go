package allegiances_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &allegiances.StoreContractSuite{
		NewStore: func() allegiances.Store { return allegiances.NewMemoryStore() },
		Account: func(n int) allegiances.Account {
			return allegiances.Account(fmt.Sprintf("01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e%02d", n))
		},
	})
}

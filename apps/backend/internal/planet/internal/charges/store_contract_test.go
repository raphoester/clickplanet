package charges_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &charges.StoreContractSuite{
		NewStore: func() charges.Store { return charges.NewMemoryStore() },
		Holder: func(n int) charges.Holder {
			return charges.Holder(fmt.Sprintf("01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e%02d", n))
		},
	})
}

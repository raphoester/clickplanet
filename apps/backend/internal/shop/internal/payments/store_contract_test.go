package payments_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &payments.StoreContractSuite{
		NewStore: func() payments.Store { return payments.NewMemoryStore() },
	})
}

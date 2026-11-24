package renewals_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &renewals.StoreContractSuite{
		NewStore: func() renewals.Store { return renewals.NewMemoryStore() },
	})
}

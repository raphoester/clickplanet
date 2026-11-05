package refusals_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/refusals"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &refusals.StoreContractSuite{
		NewStore: func() refusals.Store { return refusals.NewMemoryStore() },
	})
}

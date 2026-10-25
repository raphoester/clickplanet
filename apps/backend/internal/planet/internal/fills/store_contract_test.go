package fills_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &fills.StoreContractSuite{
		NewStore: func() fills.Store { return fills.NewMemoryStore() },
	})
}

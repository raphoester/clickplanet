package convictions_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/convictions"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &convictions.StoreContractSuite{
		NewStore: func() convictions.Store { return convictions.NewMemoryStore() },
		Account:  func(n int) string { return fmt.Sprintf("account-%d", n) },
	})
}

package shadowban_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/shadowban"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &shadowban.StoreContractSuite{
		NewStore: func() shadowban.Store { return shadowban.NewMemoryStore() },
		Key:      func(n int) string { return fmt.Sprintf("1.2.3.%d", n) },
	})
}

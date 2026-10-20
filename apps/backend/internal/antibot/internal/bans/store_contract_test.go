package bans_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &bans.StoreContractSuite{
		NewStore: func() bans.Store { return bans.NewMemoryStore() },
		Key:      func(n int) string { return fmt.Sprintf("1.2.3.%d", n) },
	})
}

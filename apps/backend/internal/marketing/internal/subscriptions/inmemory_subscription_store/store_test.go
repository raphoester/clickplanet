package inmemory_subscription_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/inmemory_subscription_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	subscriptions.StoreContractSuite
}

func (s *testSuite) SetupSuite() {
	s.NewStore = func() subscriptions.Store { return inmemory_subscription_store.New() }
}

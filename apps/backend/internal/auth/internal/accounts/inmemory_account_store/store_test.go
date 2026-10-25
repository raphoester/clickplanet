package inmemory_account_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/inmemory_account_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	accounts.StoreContractSuite
}

func (s *testSuite) SetupSuite() {
	var store *inmemory_account_store.Store
	s.NewStore = func() accounts.Store {
		store = inmemory_account_store.New()
		return store
	}
}

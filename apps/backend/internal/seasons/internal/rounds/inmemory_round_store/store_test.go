package inmemory_round_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/inmemory_round_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	rounds.StoreContractSuite
}

func (s *testSuite) SetupSuite() {
	var store *inmemory_round_store.Store
	s.NewStore = func() rounds.Store {
		store = inmemory_round_store.New()
		return store
	}
}

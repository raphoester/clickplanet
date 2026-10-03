package inmemory_worn_title_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing/inmemory_worn_title_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	wearing.StoreContractSuite
}

func (s *testSuite) SetupSuite() {
	s.NewStore = func() wearing.Store { return inmemory_worn_title_store.New() }
}

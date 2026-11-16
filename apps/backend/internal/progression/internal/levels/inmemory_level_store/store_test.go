package inmemory_level_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/inmemory_level_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	levels.StoreContractSuite
}

func (s *testSuite) SetupSuite() {
	s.NewStore = func() levels.Store { return inmemory_level_store.New() }
}

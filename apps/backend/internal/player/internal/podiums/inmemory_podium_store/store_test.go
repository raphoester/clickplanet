package inmemory_podium_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/podiums"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/podiums/inmemory_podium_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	podiums.StoreContractSuite
}

func (s *testSuite) SetupSuite() {
	s.NewStore = func() podiums.Store { return inmemory_podium_store.New() }
}

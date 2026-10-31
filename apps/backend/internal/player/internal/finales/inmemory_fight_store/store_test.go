package inmemory_fight_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales/inmemory_fight_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	finales.StoreContractSuite
}

func (s *testSuite) SetupSuite() {
	s.NewStore = func() finales.Store { return inmemory_fight_store.New() }
}

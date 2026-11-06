package inmemory_departure_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/departures"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/departures/inmemory_departure_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	departures.StoreContractSuite
}

func (s *testSuite) SetupSuite() {
	s.NewStore = func() departures.Store { return inmemory_departure_store.New() }
}

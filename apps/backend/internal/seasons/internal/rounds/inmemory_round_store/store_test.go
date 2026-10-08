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
	s.NewStore = func() rounds.Store { return inmemory_round_store.New() }
	s.ResultsOf = func(store rounds.Store, round rounds.Round) []rounds.Result {
		return store.(*inmemory_round_store.Store).Results(round) //nolint:forcetypeassert // NewStore built it.
	}
}

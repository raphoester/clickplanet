package inmemory_front_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/fronts"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/fronts/inmemory_front_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	fronts.StoreContractSuite
}

func (s *testSuite) SetupSuite() {
	s.NewStore = func() fronts.Store { return inmemory_front_store.New() }
	s.TallyOf = func(store fronts.Store, account players.AccountID) fronts.Tally {
		return store.(*inmemory_front_store.Store).Tally(account) //nolint:forcetypeassert // NewStore built it.
	}
}

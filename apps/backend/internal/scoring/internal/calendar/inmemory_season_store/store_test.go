package inmemory_season_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar/inmemory_season_store"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &calendar.StoreContractSuite{NewStore: func() calendar.Store { return inmemory_season_store.New() }})
}

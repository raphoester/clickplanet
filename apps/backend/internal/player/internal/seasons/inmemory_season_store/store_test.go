package inmemory_season_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons/inmemory_season_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, &seasons.StoreContractSuite{NewStore: func() seasons.Store { return inmemory_season_store.New() }})
}

package inmemory_mission_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/inmemory_mission_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, &missions.StoreContractSuite{NewStore: func() missions.Store { return inmemory_mission_store.New() }})
}

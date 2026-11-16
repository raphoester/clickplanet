package inmemory_unlock_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/unlocks"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/unlocks/inmemory_unlock_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, &unlocks.StoreContractSuite{NewStore: func() unlocks.Store { return inmemory_unlock_store.New() }})
}

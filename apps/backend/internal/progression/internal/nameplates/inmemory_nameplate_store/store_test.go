package inmemory_nameplate_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/nameplates"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/nameplates/inmemory_nameplate_store"
)

func TestInMemoryStoreHoldsTheContract(t *testing.T) {
	s := new(nameplates.StoreContractSuite)
	s.NewStore = func() nameplates.ReadableStore { return inmemory_nameplate_store.New() }
	suite.Run(t, s)
}

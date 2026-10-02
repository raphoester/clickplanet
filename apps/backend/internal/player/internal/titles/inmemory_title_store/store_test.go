package inmemory_title_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/inmemory_title_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	titles.StoreContractSuite
}

func (s *testSuite) SetupSuite() {
	s.NewStore = func() titles.Store { return inmemory_title_store.New() }
}

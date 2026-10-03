package inmemory_contribution_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/inmemory_contribution_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	standings.StoreContractSuite
}

func (s *testSuite) SetupSuite() {
	s.NewStore = func() standings.Store { return inmemory_contribution_store.New() }
}

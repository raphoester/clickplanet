package click_handler_service_test

import (
	"context"
	"testing"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/secondary/in_memory_tile_checker"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/secondary/memory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/domain/click_handler_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cplogging"
	"github.com/stretchr/testify/suite"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	storage *memory_tile_storage.Storage
	service *click_handler_service.Service
}

func (s *testSuite) SetupSuite() {
	const maxIndex = 250_000
	s.storage = memory_tile_storage.New(
		maxIndex,
		memory_tile_storage.Config{},
		cplogging.NewNopLogger(),
	)
	tileChecker := in_memory_tile_checker.New(maxIndex)
	countryChecker := cpcountries.New()
	s.service = click_handler_service.New(tileChecker, s.storage, countryChecker)
}

func (s *testSuite) TestNominalCase() {
	err := s.service.HandleClick(context.Background(), 1, "fr")
	s.Assert().NoError(err)
}

func (s *testSuite) TestTileOnZeroIndex() {
	err := s.service.HandleClick(context.Background(), 0, "fr")
	s.Assert().Error(err)
}

func (s *testSuite) TestInvalidCountry() {
	err := s.service.HandleClick(context.Background(), 10, "invalid")
	s.Assert().Error(err)
}

func (s *testSuite) TestInvalidTile() {
	err := s.service.HandleClick(context.Background(), 250_001, "fr")
	s.Assert().Error(err)
}

func (s *testSuite) TestTileOnLimit() {
	err := s.service.HandleClick(context.Background(), 250_000, "fr")
	s.Assert().NoError(err)
}

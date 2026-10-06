package click_usecase_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/stretchr/testify/suite"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	storage *inmemory_tile_storage.Storage
	useCase *click_usecase.UseCase
}

func (s *testSuite) SetupTest() {
	const maxIndex = 250_000
	s.storage = inmemory_tile_storage.New(
		maxIndex,
		inmemory_tile_storage.Config{},
		inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}),
		slog.New(slog.DiscardHandler),
	)
	tileChecker := clicks.NewBoard(maxIndex)
	countryChecker := cpcountries.New()
	s.useCase = click_usecase.New(tileChecker, s.storage, countryChecker)
}

func (s *testSuite) execute(tileID uint32, countryID string) error {
	_, err := s.useCase.Execute(context.Background(), click_usecase.In{TileID: tileID, CountryID: countryID})
	return err
}

func (s *testSuite) TestSpreadAndEncloseTogetherAreRefusedAndWriteNothing() {
	_, err := s.useCase.Execute(context.Background(),
		click_usecase.In{TileID: 77, CountryID: "fr", Spread: true, Enclose: true})

	s.ErrorIs(err, clicks.ErrBonusesTogether)
	owner, _ := s.storage.Owner(77)
	s.Empty(owner)
}

func (s *testSuite) TestOneBonusAtATimeIsAccepted() {
	for _, in := range []click_usecase.In{
		{TileID: 78, CountryID: "fr", Spread: true},
		{TileID: 79, CountryID: "fr", Enclose: true},
	} {
		_, err := s.useCase.Execute(context.Background(), in)
		s.NoError(err)
	}
}

func (s *testSuite) TestNominalCase() {
	s.NoError(s.execute(1, "fr"))
}

func (s *testSuite) TestTileOnZeroIndex() {
	s.Error(s.execute(0, "fr"))
}

func (s *testSuite) TestInvalidCountry() {
	s.Error(s.execute(10, "invalid"))
}

func (s *testSuite) TestInvalidTile() {
	s.Error(s.execute(250_001, "fr"))
}

func (s *testSuite) TestTileOnLimit() {
	s.NoError(s.execute(250_000, "fr"))
}

func (s *testSuite) TestAClickReachesTheFeedAsAClickedUpdate() {
	s.Require().NoError(s.execute(153, "pl"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	feed, err := s.storage.Subscribe(ctx)
	s.Require().NoError(err)

	s.Require().NoError(s.execute(153, "de"))

	change := <-feed
	s.Require().NotNil(change.Update)
	s.Equal(clicks.TileUpdate{Tile: 153, Value: "de", Previous: "pl", Clicked: true}, *change.Update)
}

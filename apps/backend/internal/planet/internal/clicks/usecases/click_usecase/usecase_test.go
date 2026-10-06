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
	s.useCase = click_usecase.New(tileChecker, s.storage, countryChecker, clicks.NewDefence(s.storage))
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

func (s *testSuite) click(tileID uint32, countryID string) click_usecase.Out {
	out, err := s.useCase.Execute(context.Background(), click_usecase.In{TileID: tileID, CountryID: countryID})
	s.Require().NoError(err)
	return out
}

func (s *testSuite) owner(tile uint32) string {
	owner, _ := s.storage.Owner(tile)
	return owner
}

func (s *testSuite) defend(tile uint32, country string, defenders int) {
	for range defenders {
		s.Require().NoError(s.storage.Reinforce(context.Background(), tile, country, 10))
	}
}

func (s *testSuite) TestEachForeignClickOnADefendedTileTakesOneDefender() {
	s.click(150, "pl")
	s.defend(150, "pl", 2)

	s.Equal(clicks.Defended, s.click(150, "de").Outcome)
	s.Equal(clicks.Defended, s.click(150, "fr").Outcome)
	s.Equal("pl", s.owner(150), "a defended tile keeps its flag")

	s.Equal(clicks.Taken, s.click(150, "de").Outcome)
	s.Equal("de", s.owner(150), "with no defender left the next click takes it")
}

func (s *testSuite) TestItsOwnFlagClickingADefendedTileChangesNothing() {
	s.click(152, "pl")
	s.defend(152, "pl", 1)

	s.Equal(clicks.Unchanged, s.click(152, "pl").Outcome)
	s.Equal(1, s.storage.Defenders(152))
}

func (s *testSuite) TestAnUndefendedTileIsTakenInOneClick() {
	s.click(300, "pl")
	s.Equal(clicks.Taken, s.click(300, "de").Outcome)
	s.Equal("de", s.owner(300))
}

func (s *testSuite) TestADefendedClickPublishesTheDefendersLeftAndNoNewOwner() {
	s.click(153, "pl")
	s.defend(153, "pl", 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	feed, err := s.storage.Subscribe(ctx)
	s.Require().NoError(err)

	s.click(153, "de")

	change := <-feed
	s.Require().NotNil(change.Update)
	s.Equal(clicks.TileUpdate{Tile: 153, Value: "pl", Previous: "pl"}, *change.Update)
	s.Empty(feed)
}

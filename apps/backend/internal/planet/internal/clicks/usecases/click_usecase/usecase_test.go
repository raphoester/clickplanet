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

type grounds struct{}

func (grounds) CountryOf(tile uint32) string {
	if tile >= 100 && tile < 200 {
		return "pl"
	}
	return ""
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
	homeSoil := clicks.NewHomeSoil(clicks.HomeSoilConfig{Enabled: true}, grounds{})
	s.useCase = click_usecase.New(tileChecker, s.storage, countryChecker, homeSoil)
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

func (s *testSuite) TestAForeignClickOnANativeTileClearsIt() {
	s.Equal(clicks.Taken, s.click(150, "pl").Outcome)

	s.Equal(clicks.Cleared, s.click(150, "de").Outcome)
	s.Empty(s.owner(150), "the first foreign click clears the tile, it does not take it")

	s.Equal(clicks.Taken, s.click(150, "de").Outcome)
	s.Equal("de", s.owner(150), "the next click takes the empty tile")
}

func (s *testSuite) TestNativesTakeBackTheirGroundInOneClick() {
	s.click(151, "pl")
	s.click(151, "de")
	s.click(151, "de")
	s.Require().Equal("de", s.owner(151))

	s.Equal(clicks.Taken, s.click(151, "pl").Outcome)
	s.Equal("pl", s.owner(151))
}

func (s *testSuite) TestNativesClickingTheirOwnTileChangeNothing() {
	s.click(152, "pl")

	s.Equal(clicks.Unchanged, s.click(152, "pl").Outcome)
	s.Equal("pl", s.owner(152))
}

func (s *testSuite) TestForeignGroundIsTakenInOneClickAsAlways() {
	s.click(300, "pl")

	s.Equal(clicks.Taken, s.click(300, "de").Outcome)
	s.Equal("de", s.owner(300))
}

func (s *testSuite) TestAClearReachesTheFeedAsAnUpdateWithNoCountry() {
	s.click(153, "pl")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	feed, err := s.storage.Subscribe(ctx)
	s.Require().NoError(err)

	s.click(153, "de")

	change := <-feed
	s.Require().NotNil(change.Update, "a clear is an ordinary update, never a blast")
	s.Equal(clicks.TileUpdate{Tile: 153, Value: "", Previous: "pl", Clicked: true}, *change.Update)
}

package postgres_tile_store_test

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
)

func (s *testSuite) TestAReplaceWritesTheMapItTakesInPlaceOfTheOneStored() {
	ctx := context.Background()
	s.Require().NoError(s.store.Save(ctx, owned("it", 1, 2, 9), []inmemory_tile_storage.Landmass{{ID: 2, Asset: "borders.bin", FortifiedBy: "it"}}))

	replaced, err := s.store.Replace(ctx, func(context.Context) ([]inmemory_tile_storage.Tile, []inmemory_tile_storage.Landmass, bool, error) {
		return append(owned("fr", 1, 2), owned("de", 3)...), []inmemory_tile_storage.Landmass{{ID: 1, Asset: "borders.bin", FortifiedBy: "fr"}}, true, nil
	})

	s.Require().NoError(err)
	s.True(replaced)
	s.Equal(map[uint32]string{1: "fr", 2: "fr", 3: "de"}, s.load())
	locks := map[clicks.LandmassID]string{}
	s.Require().NoError(s.store.LoadLandmasses(ctx, "borders.bin", func(id clicks.LandmassID, by string) { locks[id] = by }))
	s.Equal(map[clicks.LandmassID]string{1: "fr"}, locks)
}

func (s *testSuite) TestAReplaceTakeSaysNoToOrFailsKeepsTheMap() {
	ctx := context.Background()
	s.Require().NoError(s.store.Save(ctx, owned("it", 1), nil))
	broken := errors.New("broken")

	replaced, err := s.store.Replace(ctx, func(context.Context) ([]inmemory_tile_storage.Tile, []inmemory_tile_storage.Landmass, bool, error) {
		return owned("fr", 2), nil, false, nil
	})
	s.Require().NoError(err)
	s.False(replaced)
	_, err = s.store.Replace(ctx, func(context.Context) ([]inmemory_tile_storage.Tile, []inmemory_tile_storage.Landmass, bool, error) {
		return owned("fr", 2), nil, true, broken
	})
	s.Require().ErrorIs(err, broken)

	s.Equal(map[uint32]string{1: "it"}, s.load())
}

package inmemory_tile_storage_test

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
)

type counts struct {
	countries  map[string]uint32
	landmasses map[clicks.LandmassID]map[string]uint32
}

func countsOf(census clicks.Census) counts {
	read := counts{countries: map[string]uint32{}, landmasses: map[clicks.LandmassID]map[string]uint32{}}
	for _, country := range census.Countries {
		read.countries[country.Flag] = country.Tiles
	}
	for _, row := range census.Landmasses {
		flags := map[string]uint32{}
		for _, flag := range row.Flags {
			flags[flag.Flag] = flag.Tiles
		}
		read.landmasses[row.Landmass] = flags
	}
	return read
}

func (s *testSuite) census(storage *inmemory_tile_storage.Storage) (clicks.MapVersion, counts) {
	version, census := storage.Census()
	return version, countsOf(census)
}

func (s *testSuite) nextCensus(ctx context.Context, listener <-chan clicks.Change) clicks.Change {
	for {
		select {
		case <-ctx.Done():
			s.T().Fatal("no census was published")
		case change := <-listener:
			if change.Census != nil {
				return change
			}
		}
	}
}

func (s *testSuite) assertTheCensusIsTheReadAfter(storage *inmemory_tile_storage.Storage, write func(ctx context.Context)) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	storage.PublishCensus()
	_, before := s.census(storage)
	listener, _, err := storage.Subscribe(ctx)
	s.Require().NoError(err)

	write(ctx)
	storage.PublishCensus()
	change := s.nextCensus(ctx, listener)
	version, after := s.census(storage)

	s.Equal(version, change.Version, "a census counts no write of its own")
	published := countsOf(*change.Census)
	for flag, tiles := range published.countries {
		s.Equal(after.countries[flag], tiles, "flag %s, at zero too", flag)
	}
	for landmass, flags := range published.landmasses {
		s.Len(flags, len(after.landmasses[landmass]), "landmass %d is whole", landmass)
		for flag, tiles := range after.landmasses[landmass] {
			s.Equal(tiles, flags[flag], "landmass %d flag %s", landmass, flag)
		}
	}
	for flag, tiles := range after.countries {
		if before.countries[flag] != tiles {
			s.Contains(published.countries, flag, "flag %s changed", flag)
		}
	}
	for flag := range before.countries {
		if _, held := after.countries[flag]; !held {
			s.Contains(published.countries, flag, "flag %s left the map: listed at zero", flag)
		}
	}
	for _, rows := range []map[clicks.LandmassID]map[string]uint32{before.landmasses, after.landmasses} {
		for landmass := range rows {
			if !equalRows(before.landmasses[landmass], after.landmasses[landmass]) {
				s.Contains(published.landmasses, landmass, "landmass %d changed", landmass)
			}
		}
	}
}

func equalRows(a, b map[string]uint32) bool {
	if len(a) != len(b) {
		return false
	}
	for flag, tiles := range a {
		if b[flag] != tiles {
			return false
		}
	}
	return true
}

func (s *testSuite) seeded() *inmemory_tile_storage.Storage {
	storage := s.newStorage(inmemory_tile_storage.Config{})
	ctx := context.Background()
	for _, tile := range island[:2] {
		s.Require().NoError(storage.Set(ctx, tile, "fr"))
	}
	s.Require().NoError(storage.Set(ctx, island[2], "de"))
	for tile := uint32(1); tile <= 20; tile++ {
		s.Require().NoError(storage.Set(ctx, tile, "it"))
	}
	return storage
}

func (s *testSuite) TestATakeIsCensusedAsTheReadAfterIt() {
	storage := s.seeded()

	s.assertTheCensusIsTheReadAfter(storage, func(ctx context.Context) {
		s.Require().NoError(storage.Click(ctx, island[2], "fr"))
	})
}

func (s *testSuite) TestABombIsCensusedAsTheReadAfterIt() {
	storage := s.seeded()

	s.assertTheCensusIsTheReadAfter(storage, func(ctx context.Context) {
		_, err := storage.Clear(ctx, clicks.Blast{Cleared: append([]uint32{1, 2, 3}, island...)})
		s.Require().NoError(err)
	})
}

func (s *testSuite) TestAnEnclosureIsCensusedAsTheReadAfterIt() {
	storage := s.seeded()

	s.assertTheCensusIsTheReadAfter(storage, func(ctx context.Context) {
		for tile := uint32(5); tile <= 15; tile++ {
			s.Require().NoError(storage.Set(ctx, tile, "jp"))
		}
	})
}

func (s *testSuite) TestAReassignIsCensusedAsTheReadAfterIt() {
	storage := s.seeded()

	s.assertTheCensusIsTheReadAfter(storage, func(ctx context.Context) {
		_, _, err := storage.Reassign(ctx, "it", "de", 0, 100)
		s.Require().NoError(err)
	})
}

func (s *testSuite) TestAFortifyChangesNoCount() {
	storage := s.seeded()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	s.Require().NoError(storage.Set(ctx, island[2], "fr"))
	storage.PublishCensus()
	listener, _, err := storage.Subscribe(ctx)
	s.Require().NoError(err)

	_, err = storage.Fortify(ctx, island[0], "fr", 10)
	s.Require().NoError(err)
	storage.PublishCensus()

	s.NotNil((<-listener).Fortification)
	s.Empty(listener, "no census: no count moved")
}

func (s *testSuite) TestTheCensusListsEveryFlagAndEveryLandmassHeld() {
	storage := s.seeded()

	_, read := s.census(storage)

	s.Equal(map[string]uint32{"fr": 2, "de": 1, "it": 20}, read.countries)
	s.Equal(map[clicks.LandmassID]map[string]uint32{1: {"fr": 2, "de": 1}}, read.landmasses,
		"tiles on no landmass count for their flag only")
}

func (s *testSuite) TestRunPublishesTheCensusOnItsOwn() {
	storage := s.newStorage(inmemory_tile_storage.Config{})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go storage.Run(ctx)
	listener, _, err := storage.Subscribe(ctx)
	s.Require().NoError(err)

	s.Require().NoError(storage.Set(ctx, island[0], "fr"))

	s.Equal(&clicks.Census{
		Countries:  []clicks.FlagTiles{{Flag: "fr", Tiles: 1}},
		Landmasses: []clicks.LandmassTiles{{Landmass: 1, Flags: []clicks.FlagTiles{{Flag: "fr", Tiles: 1}}}},
	}, s.nextCensus(ctx, listener).Census)
}

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

func countsOf(tally clicks.Tally) counts {
	read := counts{countries: map[string]uint32{}, landmasses: map[clicks.LandmassID]map[string]uint32{}}
	for _, country := range tally.Countries {
		read.countries[country.Flag] = country.Tiles
	}
	for _, row := range tally.Landmasses {
		flags := map[string]uint32{}
		for _, flag := range row.Flags {
			flags[flag.Flag] = flag.Tiles
		}
		read.landmasses[row.Landmass] = flags
	}
	return read
}

func (s *testSuite) tally(storage *inmemory_tile_storage.Storage) (clicks.MapVersion, counts) {
	version, tally := storage.Tally()
	return version, countsOf(tally)
}

func (s *testSuite) nextTally(ctx context.Context, listener <-chan clicks.Change) clicks.Change {
	for {
		select {
		case <-ctx.Done():
			s.T().Fatal("no tally was published")
		case change := <-listener:
			if change.Tally != nil {
				return change
			}
		}
	}
}

func (s *testSuite) assertTheTallyIsTheReadAfter(storage *inmemory_tile_storage.Storage, write func(ctx context.Context)) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	storage.PublishTally()
	_, before := s.tally(storage)
	listener, _, err := storage.Subscribe(ctx)
	s.Require().NoError(err)

	write(ctx)
	storage.PublishTally()
	change := s.nextTally(ctx, listener)
	version, after := s.tally(storage)

	s.Equal(version, change.Version, "a tally counts no write of its own")
	published := countsOf(*change.Tally)
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

func (s *testSuite) TestATakeIsTalliedAsTheReadAfterIt() {
	storage := s.seeded()

	s.assertTheTallyIsTheReadAfter(storage, func(ctx context.Context) {
		s.Require().NoError(storage.Click(ctx, island[2], "fr"))
	})
}

func (s *testSuite) TestABombIsTalliedAsTheReadAfterIt() {
	storage := s.seeded()

	s.assertTheTallyIsTheReadAfter(storage, func(ctx context.Context) {
		_, err := storage.Clear(ctx, clicks.Blast{Cleared: append([]uint32{1, 2, 3}, island...)})
		s.Require().NoError(err)
	})
}

func (s *testSuite) TestAnEnclosureIsTalliedAsTheReadAfterIt() {
	storage := s.seeded()

	s.assertTheTallyIsTheReadAfter(storage, func(ctx context.Context) {
		for tile := uint32(5); tile <= 15; tile++ {
			s.Require().NoError(storage.Set(ctx, tile, "jp"))
		}
	})
}

func (s *testSuite) TestAReassignIsTalliedAsTheReadAfterIt() {
	storage := s.seeded()

	s.assertTheTallyIsTheReadAfter(storage, func(ctx context.Context) {
		_, _, err := storage.Reassign(ctx, "it", "de", 0, 100)
		s.Require().NoError(err)
	})
}

func (s *testSuite) TestAFortifyChangesNoCount() {
	storage := s.seeded()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	s.Require().NoError(storage.Set(ctx, island[2], "fr"))
	storage.PublishTally()
	listener, _, err := storage.Subscribe(ctx)
	s.Require().NoError(err)

	_, err = storage.Fortify(ctx, island[0], "fr", 10)
	s.Require().NoError(err)
	storage.PublishTally()

	s.NotNil((<-listener).Fortification)
	s.Empty(listener, "no tally: no count moved")
}

func (s *testSuite) TestTheTallyListsEveryFlagAndEveryLandmassHeld() {
	storage := s.seeded()

	_, read := s.tally(storage)

	s.Equal(map[string]uint32{"fr": 2, "de": 1, "it": 20}, read.countries)
	s.Equal(map[clicks.LandmassID]map[string]uint32{1: {"fr": 2, "de": 1}}, read.landmasses,
		"tiles on no landmass count for their flag only")
}

func (s *testSuite) TestRunPublishesTheTallyOnItsOwn() {
	storage := s.newStorage(inmemory_tile_storage.Config{})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go storage.Run(ctx)
	listener, _, err := storage.Subscribe(ctx)
	s.Require().NoError(err)

	s.Require().NoError(storage.Set(ctx, island[0], "fr"))

	s.Equal(&clicks.Tally{
		Countries:  []clicks.FlagTiles{{Flag: "fr", Tiles: 1}},
		Landmasses: []clicks.LandmassTiles{{Landmass: 1, Flags: []clicks.FlagTiles{{Flag: "fr", Tiles: 1}}}},
	}, s.nextTally(ctx, listener).Tally)
}

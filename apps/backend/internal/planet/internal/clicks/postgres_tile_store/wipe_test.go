package postgres_tile_store_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets/postgres_reset_store"
)

func (s *testSuite) TestAWipeDeletesTheMapInTheResetsTransaction() {
	ctx := context.Background()
	s.Require().NoError(s.store.Save(ctx, owned("fr", 1, 2, 3), []inmemory_tile_storage.Landmass{{ID: 1, Asset: "borders.bin", FortifiedBy: "fr"}}))
	kept := postgres_reset_store.New(s.db)
	seasonOne, err := resets.New("season-1", time.Date(2026, 11, 1, 15, 0, 0, 0, time.UTC))
	s.Require().NoError(err)
	keep := func(ctx context.Context) (bool, error) { return kept.Keep(ctx, seasonOne) }

	wiped, err := s.store.Wipe(ctx, keep)
	s.Require().NoError(err)
	s.True(wiped)
	s.Empty(s.load())
	locks := map[clicks.LandmassID]string{}
	s.Require().NoError(s.store.LoadLandmasses(ctx, "borders.bin", func(id clicks.LandmassID, by string) { locks[id] = by }))
	s.Empty(locks)

	s.Require().NoError(s.store.Save(ctx, owned("de", 7), nil))
	wiped, err = s.store.Wipe(ctx, keep)
	s.Require().NoError(err)
	s.False(wiped, "a reset kept already wipes nothing")
	s.Equal(map[uint32]string{7: "de"}, s.load())
}

func (s *testSuite) TestAWipeThatFailsKeepsTheMapAndNoReset() {
	ctx := context.Background()
	s.Require().NoError(s.store.Save(ctx, owned("fr", 1), nil))
	kept := postgres_reset_store.New(s.db)
	seasonOne, err := resets.New("season-1", time.Date(2026, 11, 1, 15, 0, 0, 0, time.UTC))
	s.Require().NoError(err)
	broken := errors.New("broken")

	_, err = s.store.Wipe(ctx, func(ctx context.Context) (bool, error) {
		if _, err := kept.Keep(ctx, seasonOne); err != nil {
			return false, fmt.Errorf("keep: %w", err)
		}
		return false, broken
	})

	s.Require().ErrorIs(err, broken)
	s.Equal(map[uint32]string{1: "fr"}, s.load())
	after, err := kept.After(ctx, time.Time{})
	s.Require().NoError(err)
	s.Empty(after, "the reset rolled back with the wipe")
}

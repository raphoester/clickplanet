//go:build testing

package inmemory_tile_storage

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func (s *Storage) Set(ctx context.Context, tile uint32, value string) error {
	return clicks.Seed(ctx, s, tile, value) //nolint:wrapcheck // a test's seed.
}

func (s *Storage) Click(ctx context.Context, tile uint32, value string) error {
	return clicks.SeedClick(ctx, s, tile, value) //nolint:wrapcheck // a test's seed.
}

func (s *Storage) Fortify(ctx context.Context, tile uint32, flag string, most int) (clicks.Fortification, error) {
	return clicks.Fortify(ctx, s, tile, flag, most) //nolint:wrapcheck // a test's seed.
}

//go:build testing

package planet

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

// Migrate makes planet's schema before the module boots, so a test stack can write the rule set it plays.
func Migrate(ctx context.Context, database cppg.Config) error {
	db := cppg.New(database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect to the planet's postgres: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Migrate(ctx, migrations.FS); err != nil {
		return fmt.Errorf("failed to migrate the %s schema: %w", database.Schema, err)
	}
	// A test binary carries the fixture beside the boards shipped, and the boot refuses a board planet keeps no row of.
	if err := eras.KeepFixture(ctx, db); err != nil {
		return fmt.Errorf("failed to keep the fixture board: %w", err)
	}
	return nil
}

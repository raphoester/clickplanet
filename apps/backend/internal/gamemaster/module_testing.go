//go:build testing

package gamemaster

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

// Migrate makes the gamemaster schema before the module boots, so a test stack can write the season it runs.
func Migrate(ctx context.Context, database cppg.Config) error {
	db := cppg.New(database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect to the gamemaster module's postgres: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Migrate(ctx, migrations.FS); err != nil {
		return fmt.Errorf("failed to migrate the %s schema: %w", database.Schema, err)
	}
	return nil
}

//go:build testing

package scoring

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

// Migrate makes the scoring schema before the module boots, so a test stack can write the season it runs.
func Migrate(ctx context.Context, database cppg.Config) error {
	db := cppg.New(database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect to the scoring module's postgres: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Migrate(ctx, migrations.FS); err != nil {
		return fmt.Errorf("failed to migrate the %s schema: %w", database.Schema, err)
	}
	return nil
}

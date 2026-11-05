//go:build testing

package antibot

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

// Migrate makes the antibot's schema before the module boots, so a test stack can write the rule set it plays.
func Migrate(ctx context.Context, database cppg.Config) error {
	db := cppg.New(database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect to the antibot's postgres: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Migrate(ctx, migrations.FS); err != nil {
		return fmt.Errorf("failed to migrate the %s schema: %w", database.Schema, err)
	}
	return nil
}

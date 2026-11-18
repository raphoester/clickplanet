//go:build testing

package cppg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/lib/pq"
)

// MigrateTo runs the migrations up or down to version, so a test of one migration starts from the version before it,
// seeds the rows it converts, and runs it.
func (p *Postgres) MigrateTo(ctx context.Context, migrations fs.FS, version uint) error {
	if _, err := p.sqlClient.ExecContext(ctx, "CREATE SCHEMA IF NOT EXISTS "+pq.QuoteIdentifier(p.config.Schema)); err != nil {
		return fmt.Errorf("failed to create schema %s: %w", p.config.Schema, err)
	}

	source, err := iofs.New(migrations, ".")
	if err != nil {
		return fmt.Errorf("failed to create iofs source: %w", err)
	}
	defer func() { _ = source.Close() }()

	m, err := migrate.NewWithSourceInstance("iofs", source, p.url())
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Migrate(version); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to migrate schema %s to %d: %w", p.config.Schema, version, err)
	}

	return nil
}

func (p *Postgres) Purge(ctx context.Context) error {
	rows, err := p.sqlClient.QueryContext(ctx, `
		SELECT quote_ident(table_schema) || '.' || quote_ident(table_name)
		FROM information_schema.tables
		WHERE table_schema = $1
		  AND table_type = 'BASE TABLE'
		  AND table_name <> 'schema_migrations'
	`, p.config.Schema)
	if err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}

	if len(tables) == 0 {
		return nil
	}

	if _, err := p.sqlClient.ExecContext(ctx, "TRUNCATE TABLE "+strings.Join(tables, ", ")+" CASCADE"); err != nil {
		return fmt.Errorf("failed to empty the tables: %w", err)
	}
	return nil
}

func (p *Postgres) Stats() sql.DBStats {
	return p.sqlClient.Stats()
}

//go:build testing

package cppg

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

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

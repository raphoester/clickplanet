// Package postgres_ban_store keeps the shadow bans between boots: one row per scope ever banned.
package postgres_ban_store

import (
	"context"
	"fmt"
	"time"

	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/shadowban"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var _ shadowban.Persistence = (*Store)(nil)

func New(db cppg.Querier) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Querier
}

// Load calls visit once per stored scope, in no particular order.
func (s *Store) Load(ctx context.Context, visit func(record shadowban.Record)) error {
	rows, err := s.db.QueryContext(ctx, `SELECT scope, flags, offences, banned_until FROM bans`)
	if err != nil {
		return fmt.Errorf("failed to read bans: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var record shadowban.Record
		if err := rows.Scan(&record.Scope, &record.Flags, &record.Offences, &record.Until); err != nil {
			return fmt.Errorf("failed to scan a ban: %w", err)
		}
		visit(record)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read bans: %w", err)
	}

	return nil
}

// Save upserts every record in one statement, so a failed save writes none.
func (s *Store) Save(ctx context.Context, records []shadowban.Record) error {
	scopes := make([]string, len(records))
	flags := make([]int64, len(records))
	offences := make([]int64, len(records))
	until := make([]time.Time, len(records))
	for i, record := range records {
		scopes[i] = record.Scope
		flags[i] = int64(record.Flags)
		offences[i] = int64(record.Offences)
		until[i] = record.Until
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO bans (scope, flags, offences, banned_until)
		SELECT * FROM unnest($1::text[], $2::integer[], $3::integer[], $4::timestamptz[])
		ON CONFLICT (scope) DO UPDATE SET
			flags = EXCLUDED.flags,
			offences = EXCLUDED.offences,
			banned_until = EXCLUDED.banned_until
	`, pq.Array(scopes), pq.Array(flags), pq.Array(offences), pq.Array(until)); err != nil {
		return fmt.Errorf("failed to upsert bans: %w", err)
	}

	return nil
}

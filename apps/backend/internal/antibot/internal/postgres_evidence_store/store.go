// Package postgres_evidence_store keeps the watchdogs' and the jury's evidence between boots: one row per section.
package postgres_evidence_store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var _ evidence.Persistence = (*Store)(nil)

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

// Load answers every stored section, saved at the latest flush that wrote one.
func (s *Store) Load(ctx context.Context) (evidence.Snapshot, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT section, data, saved_at FROM evidence`)
	if err != nil {
		return evidence.Snapshot{}, fmt.Errorf("failed to read evidence: %w", err)
	}
	defer func() { _ = rows.Close() }()

	snapshot := evidence.Snapshot{Sections: map[string][]byte{}}
	for rows.Next() {
		var (
			section string
			data    []byte
			savedAt time.Time
		)
		if err := rows.Scan(&section, &data, &savedAt); err != nil {
			return evidence.Snapshot{}, fmt.Errorf("failed to scan a section: %w", err)
		}
		snapshot.Sections[section] = data
		if savedAt.After(snapshot.SavedAt) {
			snapshot.SavedAt = savedAt
		}
	}

	if err := rows.Err(); err != nil {
		return evidence.Snapshot{}, fmt.Errorf("failed to read evidence: %w", err)
	}

	return snapshot, nil
}

// Save replaces every stored section with the snapshot's in one transaction: a section left out is deleted.
func (s *Store) Save(ctx context.Context, snapshot evidence.Snapshot) error {
	names := make([]string, 0, len(snapshot.Sections))
	for name := range snapshot.Sections {
		names = append(names, name)
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to save evidence: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM evidence WHERE section <> ALL($1::text[])`, pq.Array(names)); err != nil {
		return fmt.Errorf("failed to delete sections: %w", err)
	}

	// One statement per section: a section can be megabytes, and an array of them would hold every one twice.
	for _, name := range names {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO evidence (section, data, saved_at) VALUES ($1, $2, $3)
			ON CONFLICT (section) DO UPDATE SET data = EXCLUDED.data, saved_at = EXCLUDED.saved_at
		`, name, snapshot.Sections[name], snapshot.SavedAt); err != nil {
			return fmt.Errorf("failed to upsert the %s section: %w", name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit evidence: %w", err)
	}

	return nil
}

package postgres_snapshot_store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats/snapshots"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

var _ snapshots.Storage = (*Store)(nil)

// In one transaction: the summary and the rows of a part, or none of them.
func (s *Store) Keep(ctx context.Context, part snapshots.Part) (err error) {
	kept := part.TakeSnapshot()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin keeping a snapshot: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO snapshots (taken_at, summary) VALUES ($1, $2)
		ON CONFLICT (taken_at) DO UPDATE SET summary = EXCLUDED.summary
	`, kept.TakenAt, string(kept.Summary)); err != nil {
		return fmt.Errorf("failed to keep the summary of a snapshot: %w", err)
	}

	for _, row := range kept.Rows {
		if err := keepRow(ctx, tx, kept, row); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to keep a snapshot: %w", err)
	}
	return nil
}

func keepRow(ctx context.Context, tx *sql.Tx, kept snapshots.PartSnapshot, row snapshots.Row) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO snapshot_rows (taken_at, post_id, platform, document) VALUES ($1, $2, $3, $4)
		ON CONFLICT (taken_at, post_id, platform) DO UPDATE SET document = EXCLUDED.document
	`, kept.TakenAt, row.PostID, row.Platform, string(row.Document)); err != nil {
		return fmt.Errorf("failed to keep the row of %s on %s: %w", row.PostID, row.Platform, err)
	}
	return nil
}

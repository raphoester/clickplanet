package postgres_place_report_store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/places"
)

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

var _ places.Storage = (*Store)(nil)

func (s *Store) Save(ctx context.Context, saved ...places.Report) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin saving place reports: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, report := range saved {
		kept := report.TakeSnapshot()
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO place_reports (id, reporter_account_id, tile_id, reason, text, created_at)
			VALUES ($1, $2, $3, $4::place_report_reason, $5, $6)
			ON CONFLICT (id) DO UPDATE SET
				reporter_account_id = EXCLUDED.reporter_account_id, tile_id = EXCLUDED.tile_id,
				reason = EXCLUDED.reason, text = EXCLUDED.text, created_at = EXCLUDED.created_at
		`,
			uuid.UUID(kept.ID), reporterOf(kept.Reporter), int64(kept.Tile),
			string(kept.Complaint.Reason), kept.Complaint.Text, kept.At.UTC(),
		); err != nil {
			return fmt.Errorf("failed to upsert a place report: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the place reports saved: %w", err)
	}
	return nil
}

func reporterOf(account places.AccountID) uuid.NullUUID {
	return uuid.NullUUID{UUID: uuid.UUID(account), Valid: account != places.AccountID{}}
}

func (s *Store) FiledBy(ctx context.Context, reporter places.AccountID) ([]places.Report, error) {
	return s.restored(ctx, `
		SELECT id, reporter_account_id, tile_id, reason, text, created_at
		FROM place_reports
		WHERE reporter_account_id = $1
		ORDER BY created_at, id
	`, uuid.UUID(reporter))
}

func (s *Store) WithReporterBefore(ctx context.Context, cutoff time.Time, limit int) ([]places.Report, error) {
	return s.restored(ctx, `
		SELECT id, reporter_account_id, tile_id, reason, text, created_at
		FROM place_reports
		WHERE reporter_account_id IS NOT NULL AND created_at < $1
		ORDER BY created_at, id
		LIMIT $2
	`, cutoff.UTC(), limit)
}

func (s *Store) restored(ctx context.Context, query string, args ...any) ([]places.Report, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to read place reports: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var found []places.Report
	for rows.Next() {
		var (
			id           uuid.UUID
			reporter     uuid.NullUUID
			tile         int64
			reason, text string
			at           time.Time
		)
		if err := rows.Scan(&id, &reporter, &tile, &reason, &text, &at); err != nil {
			return nil, fmt.Errorf("failed to read a place report: %w", err)
		}

		snapshot := places.Snapshot{
			ID: places.ReportID(id), Tile: places.TileID(tile), At: at.UTC(), //nolint:gosec // a tile id, kept from a uint32.
			Complaint: places.Complaint{Reason: places.Reason(reason), Text: text},
		}
		if reporter.Valid {
			snapshot.Reporter = places.AccountID(reporter.UUID)
		}
		report, err := snapshot.Restore()
		if err != nil {
			return nil, fmt.Errorf("failed to restore the place report %s: %w", id, err)
		}
		found = append(found, report)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read place reports: %w", err)
	}
	return found, nil
}

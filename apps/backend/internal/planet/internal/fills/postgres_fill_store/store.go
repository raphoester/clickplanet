package postgres_fill_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Querier) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Querier
}

var _ fills.Store = (*Store)(nil)

const columns = `request_id, source::text, started_at, reached, accounts_filled, finished_at`

func (s *Store) Fill(ctx context.Context, request bonuses.FillID) (fills.Fill, error) {
	return keptOf(s.db.QueryRowContext(ctx, `SELECT `+columns+` FROM charge_fills WHERE request_id = $1`, string(request)))
}

func (s *Store) Unfinished(ctx context.Context) (fills.Fill, error) {
	return keptOf(s.db.QueryRowContext(ctx, `
		SELECT `+columns+` FROM charge_fills WHERE finished_at IS NULL ORDER BY started_at, request_id LIMIT 1
	`))
}

func (s *Store) Save(ctx context.Context, fill fills.Fill) error {
	kept := fill.TakeSnapshot()
	reached := uuid.NullUUID{}
	if kept.Reached != "" {
		id, err := uuid.Parse(string(kept.Reached))
		if err != nil {
			return fmt.Errorf("fill %q reached %q, which is not an account: %w", kept.ID, kept.Reached, err)
		}
		reached = uuid.NullUUID{UUID: id, Valid: true}
	}
	finished := sql.NullTime{Time: kept.FinishedAt.UTC(), Valid: !kept.FinishedAt.IsZero()}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO charge_fills (request_id, source, started_at, reached, accounts_filled, finished_at)
		VALUES ($1, $2::charge_fill_source, $3, $4, $5, $6)
		ON CONFLICT (request_id) DO UPDATE SET
			source = EXCLUDED.source,
			started_at = EXCLUDED.started_at,
			reached = EXCLUDED.reached,
			accounts_filled = EXCLUDED.accounts_filled,
			finished_at = EXCLUDED.finished_at
	`, string(kept.ID), string(kept.Source), kept.StartedAt.UTC(), reached, kept.Filled, finished); err != nil {
		return fmt.Errorf("failed to keep the fill %q: %w", kept.ID, err)
	}

	return nil
}

func keptOf(row *sql.Row) (fills.Fill, error) {
	var (
		request, source string
		startedAt       time.Time
		reached         uuid.NullUUID
		filled          int
		finished        sql.NullTime
	)
	err := row.Scan(&request, &source, &startedAt, &reached, &filled, &finished)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return fills.Fill{}, nil
	case err != nil:
		return fills.Fill{}, fmt.Errorf("failed to read a fill: %w", err)
	}

	kept := fills.Snapshot{
		ID:        bonuses.FillID(request),
		Source:    fills.Source(source),
		StartedAt: startedAt.UTC(),
		Filled:    filled,
	}
	if reached.Valid {
		kept.Reached = bonuses.Holder(reached.UUID.String())
	}
	if finished.Valid {
		kept.FinishedAt = finished.Time.UTC()
	}

	fill, err := kept.Restore()
	if err != nil {
		return fills.Fill{}, fmt.Errorf("the fill %q kept is not one: %w", request, err)
	}
	return fill, nil
}

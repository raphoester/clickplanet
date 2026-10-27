package postgres_season_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var _ calendar.Store = (*Store)(nil)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

const columns = `number, starts_at, ends_at, decided_at`

func (s *Store) Decision(ctx context.Context, number calendar.Number) (calendar.Decision, bool, error) {
	decision, err := scan(s.db.On(ctx).QueryRowContext(ctx, `SELECT `+columns+` FROM seasons WHERE number = $1`, int64(number)))
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return calendar.Decision{}, false, nil
	case err != nil:
		return calendar.Decision{}, false, fmt.Errorf("failed to read season %d: %w", number, err)
	}
	return decision, true, nil
}

func (s *Store) Decisions(ctx context.Context) ([]calendar.Decision, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT `+columns+` FROM seasons ORDER BY number`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the seasons: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var decisions []calendar.Decision
	for rows.Next() {
		decision, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to read a season: %w", err)
		}
		decisions = append(decisions, decision)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the seasons: %w", err)
	}
	return decisions, nil
}

func (s *Store) Save(ctx context.Context, decision calendar.Decision) error {
	window := decision.Window
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO seasons (`+columns+`) VALUES ($1, $2, $3, $4)
		ON CONFLICT (number) DO UPDATE SET starts_at = EXCLUDED.starts_at, ends_at = EXCLUDED.ends_at, decided_at = EXCLUDED.decided_at
	`, int64(window.Number), window.StartsAt.UTC(), window.EndsAt.UTC(), decision.DecidedAt.UTC()); err != nil {
		return fmt.Errorf("failed to keep season %d: %w", window.Number, err)
	}
	return nil
}

func scan(row interface{ Scan(dest ...any) error }) (calendar.Decision, error) {
	var (
		number   int64
		decision calendar.Decision
	)
	if err := row.Scan(&number, &decision.Window.StartsAt, &decision.Window.EndsAt, &decision.DecidedAt); err != nil {
		return calendar.Decision{}, err //nolint:wrapcheck // each caller names what it read.
	}
	decision.Window.Number = calendar.Number(number) //nolint:gosec // CHECK (number >= 0 AND number <= 4294967295).
	return decision, nil
}

package postgres_era_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var _ eras.Store = (*Store)(nil)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

func (s *Store) Boards(ctx context.Context) ([]eras.Board, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT id, name, hash FROM boards ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the boards: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var boards []eras.Board
	for rows.Next() {
		var board eras.Board
		if err := rows.Scan(&board.ID, &board.Name, &board.Hash); err != nil {
			return nil, fmt.Errorf("failed to read a board: %w", err)
		}
		boards = append(boards, board)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the boards: %w", err)
	}
	return boards, nil
}

const selectEras = `SELECT e.map, e.started_at, b.id, b.name, b.hash FROM eras e JOIN boards b ON b.id = e.board`

func (s *Store) Latest(ctx context.Context, played clicks.MapID) (eras.Era, error) {
	era, err := scanEra(s.db.On(ctx).QueryRowContext(ctx,
		selectEras+` WHERE e.map = $1 ORDER BY e.started_at DESC LIMIT 1`, string(played)))
	if errors.Is(err, sql.ErrNoRows) {
		return eras.Era{}, fmt.Errorf("%w: %s", eras.ErrNoEra, played)
	}
	if err != nil {
		return eras.Era{}, fmt.Errorf("failed to read the era of map %s: %w", played, err)
	}
	return era, nil
}

func (s *Store) Covering(ctx context.Context, played clicks.MapID, since, until time.Time) ([]eras.Era, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, selectEras+`
		WHERE e.map = $1 AND e.started_at < $3
			AND e.started_at >= COALESCE((SELECT max(started_at) FROM eras WHERE map = $1 AND started_at <= $2), $2)
		ORDER BY e.started_at
	`, string(played), since, until)
	if err != nil {
		return nil, fmt.Errorf("failed to read the eras of map %s: %w", played, err)
	}
	defer func() { _ = rows.Close() }()

	var covering []eras.Era
	for rows.Next() {
		era, err := scanEra(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to read an era of map %s: %w", played, err)
		}
		covering = append(covering, era)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the eras of map %s: %w", played, err)
	}
	return covering, nil
}

func (s *Store) Start(ctx context.Context, played clicks.Stage, at time.Time) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO eras (map, board, started_at) VALUES ($1, $2, $3) ON CONFLICT (map, started_at) DO NOTHING
	`, string(played.Map), played.Board, at); err != nil {
		return fmt.Errorf("failed to keep the era of map %s on board %d: %w", played.Map, played.Board, err)
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanEra(row scanner) (eras.Era, error) {
	var era eras.Era
	if err := row.Scan(&era.Map, &era.StartedAt, &era.Board.ID, &era.Board.Name, &era.Board.Hash); err != nil {
		return eras.Era{}, err //nolint:wrapcheck // each caller names what it read.
	}
	era.StartedAt = era.StartedAt.UTC()
	return era, nil
}

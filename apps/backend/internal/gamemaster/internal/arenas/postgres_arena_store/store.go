package postgres_arena_store

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var _ arenas.Store = (*Store)(nil)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

func (s *Store) KeepJudged(ctx context.Context, judgement moments.Judgement) (bool, error) {
	mode, err := modeOf(judgement.Mode)
	if err != nil {
		return false, err
	}
	res, err := s.db.On(ctx).ExecContext(ctx,
		`INSERT INTO battle_modes (round_id, mode) VALUES ($1, $2::battle_mode) ON CONFLICT (round_id) DO NOTHING`,
		string(judgement.Round), mode)
	if err != nil {
		return false, fmt.Errorf("failed to keep battle %s judged: %w", judgement.Round, err)
	}
	return inserted(res)
}

func (s *Store) Judged(ctx context.Context) ([]moments.Judgement, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT round_id, mode::text FROM battle_modes`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the battles judged: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var judged []moments.Judgement
	for rows.Next() {
		var round, kept string
		if err := rows.Scan(&round, &kept); err != nil {
			return nil, fmt.Errorf("failed to read a battle judged: %w", err)
		}
		mode, err := modeFrom(kept)
		if err != nil {
			return nil, err
		}
		judged = append(judged, moments.Judgement{Round: moments.RoundID(round), Mode: mode})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the battles judged: %w", err)
	}
	return judged, nil
}

func (s *Store) KeepClosed(ctx context.Context, arena moments.RoundID) (bool, error) {
	res, err := s.db.On(ctx).ExecContext(ctx,
		`INSERT INTO closed_arenas (round_id) VALUES ($1) ON CONFLICT (round_id) DO NOTHING`, string(arena))
	if err != nil {
		return false, fmt.Errorf("failed to keep arena %s closed: %w", arena, err)
	}
	return inserted(res)
}

func (s *Store) Closed(ctx context.Context) ([]moments.RoundID, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT round_id FROM closed_arenas`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the arenas closed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var closed []moments.RoundID
	for rows.Next() {
		var round string
		if err := rows.Scan(&round); err != nil {
			return nil, fmt.Errorf("failed to read an arena closed: %w", err)
		}
		closed = append(closed, moments.RoundID(round))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the arenas closed: %w", err)
	}
	return closed, nil
}

func modeOf(mode moments.Mode) (string, error) {
	switch mode {
	case moments.Plain:
		return "plain", nil
	case moments.Arena:
		return "arena", nil
	}
	return "", fmt.Errorf("no battle mode %d", mode)
}

func modeFrom(kept string) (moments.Mode, error) {
	switch kept {
	case "plain":
		return moments.Plain, nil
	case "arena":
		return moments.Arena, nil
	}
	return 0, fmt.Errorf("a battle mode this build does not know: %q", kept)
}

func inserted(res interface{ RowsAffected() (int64, error) }) (bool, error) {
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to count the rows kept: %w", err)
	}
	return n == 1, nil
}

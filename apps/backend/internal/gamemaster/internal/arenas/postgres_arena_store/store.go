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

func (s *Store) KeepOpened(ctx context.Context, arena moments.RoundID) (bool, error) {
	res, err := s.db.On(ctx).ExecContext(ctx,
		`INSERT INTO opened_arenas (round_id) VALUES ($1) ON CONFLICT (round_id) DO NOTHING`, string(arena))
	if err != nil {
		return false, fmt.Errorf("failed to keep arena %s opened: %w", arena, err)
	}
	return inserted(res)
}

func (s *Store) KeepClosed(ctx context.Context, arena moments.RoundID) (bool, error) {
	res, err := s.db.On(ctx).ExecContext(ctx,
		`INSERT INTO closed_arenas (round_id) VALUES ($1) ON CONFLICT (round_id) DO NOTHING`, string(arena))
	if err != nil {
		return false, fmt.Errorf("failed to keep arena %s closed: %w", arena, err)
	}
	return inserted(res)
}

func (s *Store) ArenasKept(ctx context.Context) (moments.ArenasKept, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT 'judged', round_id, mode::text FROM battle_modes
		UNION ALL
		SELECT 'opened', round_id, '' FROM opened_arenas
		UNION ALL
		SELECT 'closed', round_id, '' FROM closed_arenas
	`)
	if err != nil {
		return moments.ArenasKept{}, fmt.Errorf("failed to read the arena battles: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var kept moments.ArenasKept
	for rows.Next() {
		var kind, round, mode string
		if err := rows.Scan(&kind, &round, &mode); err != nil {
			return moments.ArenasKept{}, fmt.Errorf("failed to read an arena battle: %w", err)
		}
		switch kind {
		case "judged":
			judged, err := modeFrom(mode)
			if err != nil {
				return moments.ArenasKept{}, err
			}
			kept.Judged = append(kept.Judged, moments.Judgement{Round: moments.RoundID(round), Mode: judged})
		case "opened":
			kept.Opened = append(kept.Opened, moments.RoundID(round))
		case "closed":
			kept.Closed = append(kept.Closed, moments.RoundID(round))
		default:
			return moments.ArenasKept{}, fmt.Errorf("an arena row of a kind this build does not know: %q", kind)
		}
	}
	if err := rows.Err(); err != nil {
		return moments.ArenasKept{}, fmt.Errorf("failed to read the arena battles: %w", err)
	}
	return kept, nil
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

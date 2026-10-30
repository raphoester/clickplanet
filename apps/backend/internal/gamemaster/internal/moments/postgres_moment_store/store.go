package postgres_moment_store

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var _ moments.Store = (*Store)(nil)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

func (s *Store) KeepRound(ctx context.Context, scored moments.ScoredRound) (bool, error) {
	res, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO scored_rounds (round_id, book_id, ended_at) VALUES ($1, $2, $3) ON CONFLICT (round_id) DO NOTHING
	`, string(scored.Round), string(scored.Book), scored.EndedAt.UTC())
	if err != nil {
		return false, fmt.Errorf("failed to keep the round %s: %w", scored.Round, err)
	}
	return inserted(res)
}

func (s *Store) KeepBook(ctx context.Context, closed moments.ClosedBook) (bool, error) {
	res, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO closed_books (book_id, ended_at) VALUES ($1, $2) ON CONFLICT (book_id) DO NOTHING
	`, string(closed.Book), closed.EndedAt.UTC())
	if err != nil {
		return false, fmt.Errorf("failed to keep the book %s: %w", closed.Book, err)
	}
	return inserted(res)
}

func (s *Store) KeepOpened(ctx context.Context, round moments.RoundID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO opened_rounds (round_id) VALUES ($1) ON CONFLICT (round_id) DO NOTHING
	`, string(round)); err != nil {
		return fmt.Errorf("failed to keep the round %s opened: %w", round, err)
	}
	return nil
}

func (s *Store) KeepTold(ctx context.Context, day moments.ToldDay) (bool, error) {
	res, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO told_days (round_id, ends_at) VALUES ($1, $2)
		ON CONFLICT (round_id) DO UPDATE SET ends_at = excluded.ends_at WHERE told_days.ends_at <> excluded.ends_at
	`, string(day.Round), day.EndsAt.UTC())
	if err != nil {
		return false, fmt.Errorf("failed to keep the day %s told: %w", day.Round, err)
	}
	return inserted(res)
}

func (s *Store) Opened(ctx context.Context) ([]moments.RoundID, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT round_id FROM opened_rounds`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the rounds opened: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var opened []moments.RoundID
	for rows.Next() {
		var round string
		if err := rows.Scan(&round); err != nil {
			return nil, fmt.Errorf("failed to read a round opened: %w", err)
		}
		opened = append(opened, moments.RoundID(round))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the rounds opened: %w", err)
	}
	return opened, nil
}

func inserted(res interface{ RowsAffected() (int64, error) }) (bool, error) {
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to count what was kept: %w", err)
	}
	return n == 1, nil
}

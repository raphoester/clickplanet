package postgres_ping_store

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ pings.Store = (*Store)(nil)

func (s *Store) AttackPingsOf(ctx context.Context, victim nations.Flag, since time.Time) (pings.History, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT sent_at FROM attack_pings WHERE flag = $1 AND sent_at >= $2 ORDER BY sent_at
	`, string(victim), since.UTC())
	if err != nil {
		return nil, fmt.Errorf("failed to read the pings of %s: %w", victim, err)
	}
	defer func() { _ = rows.Close() }()

	var history pings.History
	for rows.Next() {
		var at time.Time
		if err := rows.Scan(&at); err != nil {
			return nil, fmt.Errorf("failed to read a ping of %s: %w", victim, err)
		}
		history = append(history, at.UTC())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the pings of %s: %w", victim, err)
	}
	return history, nil
}

func (s *Store) SaveAttackPing(ctx context.Context, victim nations.Flag, at time.Time) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO attack_pings (flag, sent_at) VALUES ($1, $2) ON CONFLICT DO NOTHING
	`, string(victim), at.UTC()); err != nil {
		return fmt.Errorf("failed to keep a ping of %s: %w", victim, err)
	}
	return nil
}

func (s *Store) StartPinged(ctx context.Context, key pings.StartKey) (bool, error) {
	var pinged bool
	if err := s.db.On(ctx).QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM start_pings WHERE key = $1)
	`, string(key)).Scan(&pinged); err != nil {
		return false, fmt.Errorf("failed to read whether %s was told: %w", key, err)
	}
	return pinged, nil
}

func (s *Store) SaveStartPing(ctx context.Context, key pings.StartKey, at time.Time) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO start_pings (key, sent_at) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET sent_at = EXCLUDED.sent_at
	`, string(key), at.UTC()); err != nil {
		return fmt.Errorf("failed to keep the start %s: %w", key, err)
	}
	return nil
}

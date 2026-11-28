package postgres_sold_store

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ nations.Store = (*Store)(nil)

func (s *Store) SoldFlags(ctx context.Context) ([]nations.Sold, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT flag, name, named_at FROM sold_flags ORDER BY flag`)
	if err != nil {
		return nil, fmt.Errorf("failed to list the sold flags: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var sold []nations.Sold
	for rows.Next() {
		var (
			flag, name string
			namedAt    time.Time
		)
		if err := rows.Scan(&flag, &name, &namedAt); err != nil {
			return nil, fmt.Errorf("failed to read a sold flag: %w", err)
		}
		restored, err := nations.SoldSnapshot{Flag: nations.Flag(flag), Name: name, NamedAt: namedAt}.Restore()
		if err != nil {
			return nil, fmt.Errorf("failed to restore a sold flag: %w", err)
		}
		sold = append(sold, restored)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to list the sold flags: %w", err)
	}
	return sold, nil
}

func (s *Store) SaveSold(ctx context.Context, sold nations.Sold) error {
	snapshot := sold.TakeSnapshot()
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO sold_flags (flag, name, named_at) VALUES ($1, $2, $3)
		ON CONFLICT (flag) DO UPDATE SET name = EXCLUDED.name, named_at = EXCLUDED.named_at
	`, string(snapshot.Flag), snapshot.Name, snapshot.NamedAt); err != nil {
		return fmt.Errorf("failed to upsert a sold flag: %w", err)
	}
	return nil
}

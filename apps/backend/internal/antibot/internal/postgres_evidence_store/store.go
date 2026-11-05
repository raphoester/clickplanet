package postgres_evidence_store

import (
	"context"
	"fmt"
	"time"

	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var _ evidence.Persistence = (*Store)(nil)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

func (s *Store) Load(ctx context.Context) (evidence.Snapshot, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT section, data, saved_at FROM evidence`)
	if err != nil {
		return evidence.Snapshot{}, fmt.Errorf("failed to read evidence: %w", err)
	}
	defer func() { _ = rows.Close() }()

	snapshot := evidence.Snapshot{Sections: map[string][]byte{}}
	for rows.Next() {
		var (
			section string
			data    []byte
			savedAt time.Time
		)
		if err := rows.Scan(&section, &data, &savedAt); err != nil {
			return evidence.Snapshot{}, fmt.Errorf("failed to scan a section: %w", err)
		}
		snapshot.Sections[section] = data
		if savedAt.After(snapshot.SavedAt) {
			snapshot.SavedAt = savedAt
		}
	}

	if err := rows.Err(); err != nil {
		return evidence.Snapshot{}, fmt.Errorf("failed to read evidence: %w", err)
	}

	return snapshot, nil
}

func (s *Store) Save(ctx context.Context, snapshot evidence.Snapshot) error {
	names := make([]string, 0, len(snapshot.Sections))
	for name := range snapshot.Sections {
		names = append(names, name)
	}

	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		if _, err := tx.ExecContext(ctx, `DELETE FROM evidence WHERE section <> ALL($1::text[])`, pq.Array(names)); err != nil {
			return fmt.Errorf("failed to delete sections: %w", err)
		}

		for _, name := range names {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO evidence (section, data, saved_at) VALUES ($1, $2, $3)
				ON CONFLICT (section) DO UPDATE SET data = EXCLUDED.data, saved_at = EXCLUDED.saved_at
			`, name, snapshot.Sections[name], snapshot.SavedAt); err != nil {
				return fmt.Errorf("failed to upsert the %s section: %w", name, err)
			}
		}
		return nil
	})
}

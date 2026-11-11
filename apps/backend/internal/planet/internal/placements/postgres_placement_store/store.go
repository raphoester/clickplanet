package postgres_placement_store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/placements"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var _ placements.Store = (*Store)(nil)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

func (s *Store) Find(ctx context.Context, ids ...clicks.PlacementID) ([]placements.Placement, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT id, account, taken_at FROM placements WHERE id = ANY($1::uuid[]) ORDER BY taken_at, id FOR UPDATE
	`, pq.Array(textOf(ids)))
	if err != nil {
		return nil, fmt.Errorf("failed to read %d placements: %w", len(ids), err)
	}
	defer func() { _ = rows.Close() }()

	var found []placements.Placement
	for rows.Next() {
		var (
			id, account uuid.UUID
			takenAt     time.Time
		)
		if err := rows.Scan(&id, &account, &takenAt); err != nil {
			return nil, fmt.Errorf("failed to read a placement: %w", err)
		}
		placement, err := placements.Snapshot{
			ID: clicks.PlacementID(id), Holder: charges.Holder(account.String()), TakenAt: takenAt,
		}.Restore()
		if err != nil {
			return nil, fmt.Errorf("failed to restore placement %s: %w", id, err)
		}
		found = append(found, placement)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the placements: %w", err)
	}
	return found, nil
}

func (s *Store) All(ctx context.Context) ([]clicks.PlacementID, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT id FROM placements ORDER BY taken_at, id`)
	if err != nil {
		return nil, fmt.Errorf("failed to list the placements: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var ids []clicks.PlacementID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to read a placement's id: %w", err)
		}
		ids = append(ids, clicks.PlacementID(id))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to list the placements: %w", err)
	}
	return ids, nil
}

func (s *Store) Save(ctx context.Context, saved ...placements.Placement) error {
	var (
		kept, ended []clicks.PlacementID
		accounts    []string
		takenAt     []time.Time
	)
	for _, placement := range saved {
		snapshot := placement.TakeSnapshot()
		if snapshot.Ended {
			ended = append(ended, snapshot.ID)
			continue
		}
		kept = append(kept, snapshot.ID)
		accounts = append(accounts, string(snapshot.Holder))
		takenAt = append(takenAt, snapshot.TakenAt)
	}

	return s.db.InTx(ctx, func(ctx context.Context) error { //nolint:wrapcheck // each step names itself.
		tx := s.db.On(ctx)
		if len(kept) > 0 {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO placements (id, account, taken_at)
				SELECT * FROM unnest($1::uuid[], $2::uuid[], $3::timestamptz[])
				ON CONFLICT (id) DO UPDATE SET account = EXCLUDED.account, taken_at = EXCLUDED.taken_at
			`, pq.Array(textOf(kept)), pq.Array(accounts), pq.Array(takenAt)); err != nil {
				return fmt.Errorf("failed to keep %d placements: %w", len(kept), err)
			}
		}
		if len(ended) > 0 {
			if _, err := tx.ExecContext(ctx, `DELETE FROM placements WHERE id = ANY($1::uuid[])`,
				pq.Array(textOf(ended))); err != nil {
				return fmt.Errorf("failed to end %d placements: %w", len(ended), err)
			}
		}
		return nil
	})
}

func textOf(ids []clicks.PlacementID) []string {
	text := make([]string, 0, len(ids))
	for _, id := range ids {
		text = append(text, id.String())
	}
	return text
}

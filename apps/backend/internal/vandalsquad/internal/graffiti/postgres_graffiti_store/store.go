package postgres_graffiti_store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ graffiti.Storage = (*Store)(nil)

func (s *Store) Save(ctx context.Context, saved ...graffiti.Graffiti) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		for _, flagged := range saved {
			if err := write(ctx, tx, flagged.TakeSnapshot()); err != nil {
				return err
			}
		}
		return nil
	})
}

func write(ctx context.Context, tx cppg.Querier, kept graffiti.Snapshot) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO graffiti (id, account, flag, kind, tiles, flagged_at, seen_at, link)
		VALUES ($1, $2, $3, $4::graffiti_kind, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			account = EXCLUDED.account, flag = EXCLUDED.flag, kind = EXCLUDED.kind, tiles = EXCLUDED.tiles,
			flagged_at = EXCLUDED.flagged_at, seen_at = EXCLUDED.seen_at, link = EXCLUDED.link
	`,
		uuid.UUID(kept.ID), accountOf(kept.Account), kept.Flag, string(kept.Kind), pq.Array(tilesOf(kept.Tiles)),
		kept.FlaggedAt.UTC(), kept.SeenAt.UTC(), kept.Link,
	); err != nil {
		return fmt.Errorf("failed to upsert a graffiti: %w", err)
	}
	return nil
}

func (s *Store) PaintedBy(ctx context.Context, account cpsession.AccountID) ([]graffiti.Graffiti, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT id, account, flag, kind, tiles, flagged_at, seen_at, link
		FROM graffiti
		WHERE account = $1
		ORDER BY flagged_at, id
	`, uuid.UUID(account))
	if err != nil {
		return nil, fmt.Errorf("failed to read the graffiti an account painted: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var found []graffiti.Graffiti
	for rows.Next() {
		snapshot, err := scan(rows)
		if err != nil {
			return nil, err
		}
		restored, err := snapshot.Restore()
		if err != nil {
			return nil, fmt.Errorf("failed to restore the graffiti %s: %w", uuid.UUID(snapshot.ID), err)
		}
		found = append(found, restored)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the graffiti an account painted: %w", err)
	}
	return found, nil
}

func scan(row interface{ Scan(dest ...any) error }) (graffiti.Snapshot, error) {
	var (
		id                uuid.UUID
		account           uuid.NullUUID
		flag, kind, link  string
		tiles             pq.Int64Array
		flagged, lastSeen time.Time
	)
	if err := row.Scan(&id, &account, &flag, &kind, &tiles, &flagged, &lastSeen, &link); err != nil {
		return graffiti.Snapshot{}, fmt.Errorf("failed to read a graffiti: %w", err)
	}

	ids := make([]drawings.TileID, len(tiles))
	for i, tile := range tiles {
		ids[i] = drawings.TileID(tile) //nolint:gosec // the map's ids, a few hundred thousand.
	}
	owner := cpsession.NoAccount
	if account.Valid {
		owner = cpsession.AccountID(account.UUID)
	}
	return graffiti.Snapshot{
		ID: graffiti.ID(id), Account: owner, Flag: flag, Kind: graffiti.Kind(kind), Tiles: ids,
		FlaggedAt: flagged.UTC(), SeenAt: lastSeen.UTC(), Link: link,
	}, nil
}

func tilesOf(ids []drawings.TileID) []int64 {
	tiles := make([]int64, len(ids))
	for i, id := range ids {
		tiles[i] = int64(id)
	}
	return tiles
}

func accountOf(account cpsession.AccountID) uuid.NullUUID {
	if account == cpsession.NoAccount {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: uuid.UUID(account), Valid: true}
}

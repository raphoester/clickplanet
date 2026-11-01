package postgres_podium_store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/podiums"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ podiums.Store = (*Store)(nil)

func (s *Store) Record(ctx context.Context, places []podiums.Place) error {
	if len(places) == 0 {
		return nil
	}

	var (
		seasons, ranks []int64
		accounts       []string
	)
	for _, place := range places {
		snapshot := place.TakeSnapshot()
		seasons = append(seasons, int64(snapshot.Season))
		accounts = append(accounts, snapshot.Account.String())
		ranks = append(ranks, int64(snapshot.Rank))
	}

	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO podium_places (season, account_id, rank)
		SELECT * FROM unnest($1::integer[], $2::uuid[], $3::integer[])
		ON CONFLICT (season, account_id) DO UPDATE SET rank = EXCLUDED.rank
	`, pq.Array(seasons), pq.Array(accounts), pq.Array(ranks)); err != nil {
		return fmt.Errorf("failed to record the podium places: %w", err)
	}
	return nil
}

func (s *Store) Places(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]podiums.Places, error) {
	found := map[players.AccountID]podiums.Places{}
	if len(accounts) == 0 {
		return found, nil
	}

	ids := make([]string, len(accounts))
	for i, account := range accounts {
		ids[i] = account.String()
	}

	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT account_id, season, rank FROM podium_places
		WHERE account_id = ANY($1::uuid[])
		ORDER BY account_id, season
	`, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to read the podium places: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			account      uuid.UUID
			season, rank int64
		)
		if err := rows.Scan(&account, &season, &rank); err != nil {
			return nil, fmt.Errorf("failed to read a podium place: %w", err)
		}
		place, err := podiums.PlaceSnapshot{
			Account: players.AccountID(account),
			Season:  finales.Season(season), //nolint:gosec // CHECK (season >= 0).
			Rank:    uint32(rank),           //nolint:gosec // CHECK (rank BETWEEN 1 AND 3).
		}.Restore()
		if err != nil {
			return nil, fmt.Errorf("failed to restore a podium place: %w", err)
		}
		found[players.AccountID(account)] = append(found[players.AccountID(account)], place)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the podium places: %w", err)
	}
	return found, nil
}

func (s *Store) DeleteAccount(ctx context.Context, account players.AccountID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM podium_places WHERE account_id = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to delete the account's podium places: %w", err)
	}
	return nil
}

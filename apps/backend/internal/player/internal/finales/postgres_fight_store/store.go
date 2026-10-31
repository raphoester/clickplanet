package postgres_fight_store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ finales.Store = (*Store)(nil)

func (s *Store) Record(ctx context.Context, fights []finales.Fight) error {
	if len(fights) == 0 {
		return nil
	}

	var (
		seasons, ranks []int64
		accounts       []string
		countries      []string
	)
	for _, fight := range fights {
		snapshot := fight.TakeSnapshot()
		seasons = append(seasons, int64(snapshot.Season))
		accounts = append(accounts, snapshot.Account.String())
		countries = append(countries, string(snapshot.Country))
		ranks = append(ranks, int64(snapshot.Rank))
	}

	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO finale_fights (season, account_id, country, flag_rank)
		SELECT * FROM unnest($1::integer[], $2::uuid[], $3::text[], $4::integer[])
		ON CONFLICT (season, account_id) DO NOTHING
	`, pq.Array(seasons), pq.Array(accounts), pq.Array(countries), pq.Array(ranks)); err != nil {
		return fmt.Errorf("failed to record the fights: %w", err)
	}
	return nil
}

func (s *Store) Fights(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]finales.Fights, error) {
	found := map[players.AccountID]finales.Fights{}
	if len(accounts) == 0 {
		return found, nil
	}

	ids := make([]string, len(accounts))
	for i, account := range accounts {
		ids[i] = account.String()
	}

	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT account_id, season, country, flag_rank FROM finale_fights
		WHERE account_id = ANY($1::uuid[])
		ORDER BY account_id, season
	`, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to read the fights: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			account      uuid.UUID
			season, rank int64
			country      string
		)
		if err := rows.Scan(&account, &season, &country, &rank); err != nil {
			return nil, fmt.Errorf("failed to read a fight: %w", err)
		}
		fight, err := finales.FightSnapshot{
			Account: players.AccountID(account),
			Season:  finales.Season(season), //nolint:gosec // CHECK (season >= 0).
			Country: finales.Country(country),
			Rank:    uint32(rank), //nolint:gosec // CHECK (flag_rank >= 0).
		}.Restore()
		if err != nil {
			return nil, fmt.Errorf("failed to restore a fight: %w", err)
		}
		found[players.AccountID(account)] = append(found[players.AccountID(account)], fight)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the fights: %w", err)
	}
	return found, nil
}

func (s *Store) DeleteAccount(ctx context.Context, account players.AccountID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM finale_fights WHERE account_id = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to delete the account's fights: %w", err)
	}
	return nil
}

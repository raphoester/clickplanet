package postgres_season_store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

var _ seasons.Store = (*Store)(nil)

func (s *Store) Record(ctx context.Context, account players.AccountID) (seasons.Record, error) {
	records, err := s.Records(ctx, []players.AccountID{account})
	if err != nil {
		return seasons.Record{}, err
	}
	if record, ok := records[account]; ok {
		return record, nil
	}
	return seasons.NewRecord(account), nil
}

func (s *Store) Records(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]seasons.Record, error) {
	records := map[players.AccountID]seasons.Record{}
	if len(accounts) == 0 {
		return records, nil
	}

	ids := make([]string, len(accounts))
	for i, account := range accounts {
		ids[i] = account.String()
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT account_id, season, country, won FROM seasons_played WHERE account_id = ANY($1::uuid[])
	`, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to read the seasons played: %w", err)
	}
	defer func() { _ = rows.Close() }()

	snapshots := map[players.AccountID]seasons.RecordSnapshot{}
	for rows.Next() {
		var (
			account uuid.UUID
			season  int64
			country string
			won     bool
		)
		if err := rows.Scan(&account, &season, &country, &won); err != nil {
			return nil, fmt.Errorf("failed to read a season played: %w", err)
		}
		snapshot := snapshots[players.AccountID(account)]
		snapshot.Account = players.AccountID(account)
		snapshot.Seasons = append(snapshot.Seasons, seasons.Season{
			Number: seasons.Number(season), //nolint:gosec // CHECK (season >= 0), a season a few weeks.
			Flag:   seasons.Country(country),
			Won:    won,
		})
		snapshots[players.AccountID(account)] = snapshot
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the seasons played: %w", err)
	}

	for account, snapshot := range snapshots {
		records[account] = snapshot.Restore()
	}
	return records, nil
}

func (s *Store) Save(ctx context.Context, record seasons.Record) (err error) {
	snapshot := record.TakeSnapshot()
	numbers := make([]int64, len(snapshot.Seasons))
	countries := make([]string, len(snapshot.Seasons))
	won := make([]bool, len(snapshot.Seasons))
	for i, season := range snapshot.Seasons {
		numbers[i], countries[i], won[i] = int64(season.Number), string(season.Flag), season.Won
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin saving the seasons played: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM seasons_played WHERE account_id = $1 AND NOT season = ANY($2::integer[])
	`, uuid.UUID(snapshot.Account), pq.Array(numbers)); err != nil {
		return fmt.Errorf("failed to drop the seasons no longer played: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO seasons_played (account_id, season, country, won)
		SELECT $1, played.season, played.country, played.won
		FROM unnest($2::integer[], $3::text[], $4::boolean[]) AS played (season, country, won)
		ON CONFLICT (account_id, season) DO UPDATE SET country = excluded.country, won = excluded.won
	`, uuid.UUID(snapshot.Account), pq.Array(numbers), pq.Array(countries), pq.Array(won)); err != nil {
		return fmt.Errorf("failed to save the seasons played: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the seasons played: %w", err)
	}
	return nil
}

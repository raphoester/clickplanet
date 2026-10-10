package postgres_round_store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

var _ rounds.Store = (*Store)(nil)

func (s *Store) RecordSnapshot(ctx context.Context, round rounds.Round, snapshot rounds.Snapshot) (err error) {
	countries := make([]string, 0, len(snapshot.Held))
	tiles := make([]int64, 0, len(snapshot.Held))
	for country, held := range snapshot.Held {
		if held > 0 {
			countries = append(countries, string(country))
			tiles = append(tiles, int64(held))
		}
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin counting the snapshot: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO rounds (season, ends_at, finale, samples, map_tiles) VALUES ($1, $2, $3, 1, $4)
		ON CONFLICT (season, ends_at) DO UPDATE SET samples = rounds.samples + 1, map_tiles = excluded.map_tiles
	`, int64(round.Season), round.EndsAt, round.Finale, int64(snapshot.Tiles)); err != nil {
		return fmt.Errorf("failed to count the snapshot in its round: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO round_holdings (season, ends_at, country, tiles)
		SELECT $1, $2, held.country, held.tiles FROM unnest($3::text[], $4::bigint[]) AS held (country, tiles)
		ON CONFLICT (season, ends_at, country) DO UPDATE SET tiles = round_holdings.tiles + excluded.tiles
	`, int64(round.Season), round.EndsAt, pq.Array(countries), pq.Array(tiles)); err != nil {
		return fmt.Errorf("failed to add what each country held: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the snapshot: %w", err)
	}
	return nil
}

func (s *Store) Unclosed(ctx context.Context, endedBy time.Time) ([]rounds.Round, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT season, ends_at, finale FROM rounds WHERE NOT closed AND ends_at <= $1 ORDER BY ends_at, season
	`, endedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to read the rounds left to close: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var ended []rounds.Round
	for rows.Next() {
		var (
			season int64
			round  rounds.Round
		)
		if err := rows.Scan(&season, &round.EndsAt, &round.Finale); err != nil {
			return nil, fmt.Errorf("failed to read a round left to close: %w", err)
		}
		round.Season = calendar.Number(season) //nolint:gosec // CHECK (season >= 0).
		round.EndsAt = round.EndsAt.UTC()
		ended = append(ended, round)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the rounds left to close: %w", err)
	}
	return ended, nil
}

func (s *Store) Held(ctx context.Context, round rounds.Round) (map[rounds.Country]uint64, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT country, tiles FROM round_holdings WHERE season = $1 AND ends_at = $2
	`, int64(round.Season), round.EndsAt)
	if err != nil {
		return nil, fmt.Errorf("failed to read what each country held: %w", err)
	}
	defer func() { _ = rows.Close() }()

	held := map[rounds.Country]uint64{}
	for rows.Next() {
		var (
			country string
			tiles   int64
		)
		if err := rows.Scan(&country, &tiles); err != nil {
			return nil, fmt.Errorf("failed to read what a country held: %w", err)
		}
		held[rounds.Country(country)] = uint64(tiles) //nolint:gosec // CHECK (tiles > 0).
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read what each country held: %w", err)
	}
	return held, nil
}

func (s *Store) Number(ctx context.Context, round rounds.Round) (uint32, error) {
	var earlier int64
	if err := s.db.QueryRowContext(ctx, `
		SELECT count(*) FROM rounds WHERE season = $1 AND ends_at < $2
	`, int64(round.Season), round.EndsAt).Scan(&earlier); err != nil {
		return 0, fmt.Errorf("failed to count the rounds before this one: %w", err)
	}
	return uint32(earlier + 1), nil //nolint:gosec // a round a day.
}

func (s *Store) Close(ctx context.Context, round rounds.Round, results []rounds.Result) (err error) {
	countries := make([]string, 0, len(results))
	ranks := make([]int64, 0, len(results))
	points := make([]int64, 0, len(results))
	for _, result := range results {
		countries = append(countries, string(result.Country))
		ranks = append(ranks, int64(result.Rank))
		points = append(points, int64(result.Points))
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin closing the round: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	closing, err := tx.ExecContext(ctx, `
		UPDATE rounds SET closed = true WHERE season = $1 AND ends_at = $2 AND NOT closed
	`, int64(round.Season), round.EndsAt)
	if err != nil {
		return fmt.Errorf("failed to close the round: %w", err)
	}
	closed, err := closing.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to close the round: %w", err)
	}
	if closed == 0 {
		return tx.Rollback() //nolint:wrapcheck // nothing was written.
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO round_results (season, ends_at, country, rank, points)
		SELECT $1, $2, result.country, result.rank, result.points
		FROM unnest($3::text[], $4::integer[], $5::integer[]) AS result (country, rank, points)
	`, int64(round.Season), round.EndsAt, pq.Array(countries), pq.Array(ranks), pq.Array(points)); err != nil {
		return fmt.Errorf("failed to keep the round's results: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the round's results: %w", err)
	}
	return nil
}

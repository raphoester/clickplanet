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

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ rounds.Store = (*Store)(nil)

func (s *Store) Open(ctx context.Context, round rounds.Round) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO rounds (season, ends_at, finale) VALUES ($1, $2, $3) ON CONFLICT (season, ends_at) DO NOTHING
	`, int64(round.Season), round.EndsAt, round.Finale); err != nil {
		return fmt.Errorf("failed to open the round: %w", err)
	}
	return nil
}

func (s *Store) Unclosed(ctx context.Context, endedBy time.Time) ([]rounds.Round, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `
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

func (s *Store) Number(ctx context.Context, round rounds.Round) (uint32, error) {
	var earlier int64
	if err := s.db.On(ctx).QueryRowContext(ctx, `
		SELECT count(*) FROM rounds WHERE season = $1 AND ends_at < $2
	`, int64(round.Season), round.EndsAt).Scan(&earlier); err != nil {
		return 0, fmt.Errorf("failed to count the rounds before this one: %w", err)
	}
	return uint32(earlier + 1), nil //nolint:gosec // a round a day.
}

// Close answers whether this call closed the round: one closed before writes nothing.
func (s *Store) Close(ctx context.Context, round rounds.Round, results []rounds.Result) (bool, error) {
	countries := make([]string, 0, len(results))
	ranks := make([]int64, 0, len(results))
	points := make([]int64, 0, len(results))
	shares := make([]float64, 0, len(results))
	for _, result := range results {
		countries = append(countries, string(result.Country))
		ranks = append(ranks, int64(result.Rank))
		points = append(points, int64(result.Points))
		shares = append(shares, result.Share)
	}

	closed := false
	err := s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		closing, err := tx.ExecContext(ctx, `
			UPDATE rounds SET closed = true WHERE season = $1 AND ends_at = $2 AND NOT closed
		`, int64(round.Season), round.EndsAt)
		if err != nil {
			return fmt.Errorf("failed to close the round: %w", err)
		}
		updated, err := closing.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to close the round: %w", err)
		}
		if updated == 0 {
			return nil
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO round_results (season, ends_at, country, rank, points, share)
			SELECT $1, $2, result.country, result.rank, result.points, result.share
			FROM unnest($3::text[], $4::integer[], $5::integer[], $6::double precision[]) AS result (country, rank, points, share)
		`, int64(round.Season), round.EndsAt, pq.Array(countries), pq.Array(ranks), pq.Array(points), pq.Array(shares)); err != nil {
			return fmt.Errorf("failed to keep the round's results: %w", err)
		}
		closed = true
		return nil
	})
	return closed, err
}

func (s *Store) ClosedIn(ctx context.Context, season calendar.Number) ([]rounds.Closed, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `
		WITH numbered AS (
			SELECT season, ends_at, finale, closed, row_number() OVER (ORDER BY ends_at) AS number
			FROM rounds WHERE season = $1
		)
		SELECT round.ends_at, round.finale, round.number, result.country, result.rank, result.points, result.share
		FROM numbered AS round LEFT JOIN round_results AS result USING (season, ends_at)
		WHERE round.closed
		ORDER BY round.ends_at, result.rank, result.country
	`, int64(season))
	if err != nil {
		return nil, fmt.Errorf("failed to read the rounds the season closed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var closed []rounds.Closed
	for rows.Next() {
		var (
			endsAt       time.Time
			finale       bool
			number       int64
			country      sql.NullString
			rank, points sql.NullInt64
			share        sql.NullFloat64
		)
		if err := rows.Scan(&endsAt, &finale, &number, &country, &rank, &points, &share); err != nil {
			return nil, fmt.Errorf("failed to read a result of a round the season closed: %w", err)
		}
		round := rounds.Round{Season: season, EndsAt: endsAt.UTC(), Finale: finale}
		if len(closed) == 0 || !closed[len(closed)-1].Round.EndsAt.Equal(round.EndsAt) {
			closed = append(closed, rounds.Closed{Round: round, Number: uint32(number)}) //nolint:gosec // a round a day.
		}
		if country.Valid {
			last := &closed[len(closed)-1]
			last.Results = append(last.Results, rounds.Result{
				Country: rounds.Country(country.String),
				Rank:    uint32(rank.Int64),   //nolint:gosec // a few hundred countries at most.
				Points:  uint32(points.Int64), //nolint:gosec // 75 a round at most.
				Share:   share.Float64,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the rounds the season closed: %w", err)
	}
	return closed, nil
}

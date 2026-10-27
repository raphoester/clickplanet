package postgres_round_store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var (
	_ rounds.Store = (*Store)(nil)
	_ rounds.Takes = (*Store)(nil)
)

func startOf(round rounds.Round) sql.NullTime {
	if !round.Own {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: round.StartsAt.UTC(), Valid: true}
}

// Open keeps the entry of a round of its own opened first: a second entry that ends with it changes nothing.
func (s *Store) Open(ctx context.Context, round rounds.Round) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO rounds (season, ends_at, own, finale, starts_at, times) VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (season, ends_at, own) DO NOTHING
		`, int64(round.Season), round.EndsAt, round.Own, round.Finale, startOf(round), int64(round.Times)); err != nil {
			return fmt.Errorf("failed to open the round: %w", err)
		}
		if !round.Own {
			return nil
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO own_rounds (season, ends_at, own, entry_id) VALUES ($1, $2, true, $3)
			ON CONFLICT (season, ends_at, own) DO NOTHING
		`, int64(round.Season), round.EndsAt, uuid.UUID(round.Entry)); err != nil {
			return fmt.Errorf("failed to keep the entry of the round of its own: %w", err)
		}
		return nil
	})
}

const (
	roundColumns = `season, ends_at, own, finale, starts_at, times, entry_id`
	withEntries  = `rounds LEFT JOIN own_rounds USING (season, ends_at, own)`
)

type scanner interface {
	Scan(dest ...any) error
}

func scanRound(row scanner, extra ...any) (rounds.Round, error) {
	var (
		season, times int64
		startsAt      sql.NullTime
		entry         uuid.NullUUID
		round         rounds.Round
	)
	if err := row.Scan(append([]any{&season, &round.EndsAt, &round.Own, &round.Finale, &startsAt, &times, &entry}, extra...)...); err != nil {
		return rounds.Round{}, err //nolint:wrapcheck // each caller says what it read.
	}
	round.Season = calendar.Number(season) //nolint:gosec // CHECK (season >= 0).
	round.EndsAt = round.EndsAt.UTC()
	round.Times = uint32(times) //nolint:gosec // CHECK (times BETWEEN 1 AND 10).
	if startsAt.Valid {
		round.StartsAt = startsAt.Time.UTC()
	}
	round.Entry = rounds.EntryID(entry.UUID)
	return round, nil
}

func (s *Store) Unclosed(ctx context.Context, endedBy time.Time) ([]rounds.Round, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT `+roundColumns+` FROM `+withEntries+` WHERE NOT closed AND ends_at <= $1 ORDER BY ends_at, season, own
	`, endedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to read the rounds left to close: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var ended []rounds.Round
	for rows.Next() {
		round, err := scanRound(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to read a round left to close: %w", err)
		}
		ended = append(ended, round)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the rounds left to close: %w", err)
	}
	return ended, nil
}

func (s *Store) FirstEnd(ctx context.Context) (time.Time, bool, error) {
	var first sql.NullTime
	if err := s.db.On(ctx).QueryRowContext(ctx, `SELECT min(ends_at) FROM rounds WHERE NOT closed`).Scan(&first); err != nil {
		return time.Time{}, false, fmt.Errorf("failed to read when the next round ends: %w", err)
	}
	return first.Time.UTC(), first.Valid, nil
}

func (s *Store) Number(ctx context.Context, round rounds.Round) (uint32, error) {
	var earlier int64
	if err := s.db.On(ctx).QueryRowContext(ctx, `
		SELECT count(*) FROM rounds WHERE season = $1 AND own = $2 AND ends_at < $3
	`, int64(round.Season), round.Own, round.EndsAt).Scan(&earlier); err != nil {
		return 0, fmt.Errorf("failed to count the rounds before this one: %w", err)
	}
	return uint32(earlier + 1), nil //nolint:gosec // a round a day.
}

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
			UPDATE rounds SET closed = true WHERE season = $1 AND ends_at = $2 AND own = $3 AND NOT closed
		`, int64(round.Season), round.EndsAt, round.Own)
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
			INSERT INTO round_results (season, ends_at, own, country, rank, points, share)
			SELECT $1, $2, $3, result.country, result.rank, result.points, result.share
			FROM unnest($4::text[], $5::integer[], $6::integer[], $7::double precision[]) AS result (country, rank, points, share)
		`, int64(round.Season), round.EndsAt, round.Own,
			pq.Array(countries), pq.Array(ranks), pq.Array(points), pq.Array(shares)); err != nil {
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
			SELECT `+roundColumns+`, closed, row_number() OVER (PARTITION BY own ORDER BY ends_at) AS number
			FROM `+withEntries+` WHERE season = $1
		)
		SELECT round.season, round.ends_at, round.own, round.finale, round.starts_at, round.times, round.entry_id,
		       round.number,
		       result.country, result.rank, result.points, result.share
		FROM numbered AS round LEFT JOIN round_results AS result USING (season, ends_at, own)
		WHERE round.closed
		ORDER BY round.ends_at, round.own, result.rank, result.country
	`, int64(season))
	if err != nil {
		return nil, fmt.Errorf("failed to read the rounds the season closed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var closed []rounds.Closed
	for rows.Next() {
		var (
			number       int64
			country      sql.NullString
			rank, points sql.NullInt64
			share        sql.NullFloat64
		)
		round, err := scanRound(rows, &number, &country, &rank, &points, &share)
		if err != nil {
			return nil, fmt.Errorf("failed to read a result of a round the season closed: %w", err)
		}
		if len(closed) == 0 || !sameRound(closed[len(closed)-1].Round, round) {
			closed = append(closed, rounds.Closed{Round: round, Number: uint32(number)}) //nolint:gosec // a round a day.
		}
		if country.Valid {
			last := &closed[len(closed)-1]
			last.Results = append(last.Results, rounds.Result{
				Country: rounds.Country(country.String),
				Rank:    uint32(rank.Int64),   //nolint:gosec // a few hundred countries at most.
				Points:  uint32(points.Int64), //nolint:gosec // 250 a round at most.
				Share:   share.Float64,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the rounds the season closed: %w", err)
	}
	return closed, nil
}

func sameRound(a, b rounds.Round) bool {
	return a.EndsAt.Equal(b.EndsAt) && a.Own == b.Own
}

func (s *Store) End(ctx context.Context, season calendar.Season) (bool, error) {
	ending, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO ended_seasons (season, ended_at) VALUES ($1, $2) ON CONFLICT (season) DO NOTHING
	`, int64(season.Number), season.EndsAt)
	if err != nil {
		return false, fmt.Errorf("failed to end the season: %w", err)
	}
	ended, err := ending.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to end the season: %w", err)
	}
	return ended > 0, nil
}

func (s *Store) Ended(ctx context.Context, season calendar.Number) (bool, error) {
	var ended bool
	if err := s.db.On(ctx).QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM ended_seasons WHERE season = $1)
	`, int64(season)).Scan(&ended); err != nil {
		return false, fmt.Errorf("failed to read whether the season ended: %w", err)
	}
	return ended, nil
}

// Count moves each window's mark past the take and counts it in one statement: a take told again is under the mark.
func (s *Store) Count(ctx context.Context, take rounds.Take) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		WITH counted AS (
			UPDATE rounds SET counted_takes_below = $4
			WHERE own AND starts_at <= $3 AND ends_at > $3 AND counted_takes_below < $4
			RETURNING season, ends_at, own
		), countries AS (
			INSERT INTO round_takes (season, ends_at, own, country, tiles)
			SELECT season, ends_at, own, $2, 1 FROM counted
			ON CONFLICT (season, ends_at, own, country) DO UPDATE SET tiles = round_takes.tiles + 1
		)
		INSERT INTO round_players (season, ends_at, own, account_id, country, tiles)
		SELECT season, ends_at, own, $1, $2, 1 FROM counted
		ON CONFLICT (season, ends_at, own, account_id, country) DO UPDATE SET tiles = round_players.tiles + 1
	`, uuid.UUID(take.Account), string(take.Country), take.At,
		int64(take.Position+1)); err != nil { //nolint:gosec // a ledger position fits in int64.
		return fmt.Errorf("failed to count the take in its rounds: %w", err)
	}
	return nil
}

func (s *Store) ForgetAccount(ctx context.Context, account rounds.AccountID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM round_players WHERE account_id = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to forget the account's takes in the rounds: %w", err)
	}
	return nil
}

package postgres_round_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var (
	_ books.Rounds = (*Store)(nil)
	_ books.Takes  = (*Store)(nil)
)

const roundColumns = `
	SELECT round.round_id, round.book_id, round.times, round.starts_at, scored.ended_at
	FROM rounds AS round LEFT JOIN scored_rounds AS scored USING (round_id)
`

type scanner interface {
	Scan(dest ...any) error
}

func scanRound(row scanner, extra ...any) (books.Round, error) {
	var (
		id, book string
		times    int64
		endedAt  sql.NullTime
		round    books.Round
	)
	if err := row.Scan(append([]any{&id, &book, &times, &round.StartsAt, &endedAt}, extra...)...); err != nil {
		return books.Round{}, err //nolint:wrapcheck // each caller says what it read.
	}
	round.ID, round.Book, round.StartsAt = books.RoundID(id), books.BookID(book), round.StartsAt.UTC()
	round.Times = uint32(times) //nolint:gosec // CHECK (times BETWEEN 1 AND 10).
	if endedAt.Valid {
		round.Scored, round.EndsAt = true, endedAt.Time.UTC()
	}
	return round, nil
}

func (s *Store) Find(ctx context.Context, id books.RoundID) (books.Round, bool, error) {
	round, err := scanRound(s.db.On(ctx).QueryRowContext(ctx, roundColumns+` WHERE round.round_id = $1`, string(id)))
	if errors.Is(err, sql.ErrNoRows) {
		return books.Round{}, false, nil
	}
	if err != nil {
		return books.Round{}, false, fmt.Errorf("failed to read the round: %w", err)
	}
	return round, true, nil
}

func (s *Store) Of(ctx context.Context, of []books.BookID) ([]books.Round, error) {
	ids := make([]string, 0, len(of))
	for _, book := range of {
		ids = append(ids, string(book))
	}
	rows, err := s.db.On(ctx).QueryContext(ctx, roundColumns+`
		WHERE round.book_id = ANY($1) ORDER BY round.starts_at, round.round_id
	`, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to read the rounds of the books: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var rounds []books.Round
	for rows.Next() {
		round, err := scanRound(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to read a round of the books: %w", err)
		}
		rounds = append(rounds, round)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the rounds of the books: %w", err)
	}
	return rounds, nil
}

func (s *Store) Open(ctx context.Context, round books.Round) (bool, error) {
	opening, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO rounds (round_id, book_id, times, starts_at) VALUES ($1, $2, $3, $4) ON CONFLICT (round_id) DO NOTHING
	`, string(round.ID), string(round.Book), int64(round.Times), round.StartsAt)
	if err != nil {
		return false, fmt.Errorf("failed to open the round: %w", err)
	}
	return applied(opening, "failed to open the round")
}

func applied(result sql.Result, failure string) (bool, error) {
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("%s: %w", failure, err)
	}
	return affected > 0, nil
}

func (s *Store) Score(ctx context.Context, scored books.Scored) (bool, error) {
	countries := make([]string, 0, len(scored.Results))
	ranks := make([]int64, 0, len(scored.Results))
	points := make([]int64, 0, len(scored.Results))
	shares := make([]float64, 0, len(scored.Results))
	for _, result := range scored.Results {
		countries = append(countries, string(result.Country))
		ranks = append(ranks, int64(result.Rank))
		points = append(points, int64(result.Points))
		shares = append(shares, result.Share)
	}

	done := false
	err := s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		scoring, err := tx.ExecContext(ctx, `
			INSERT INTO scored_rounds (round_id, ended_at) SELECT round_id, $2 FROM rounds WHERE round_id = $1
			ON CONFLICT (round_id) DO NOTHING
		`, string(scored.Round.ID), scored.Round.EndsAt)
		if err != nil {
			return fmt.Errorf("failed to score the round: %w", err)
		}
		if done, err = applied(scoring, "failed to score the round"); err != nil || !done {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO round_results (round_id, country, rank, points, share)
			SELECT $1, result.country, result.rank, result.points, result.share
			FROM unnest($2::text[], $3::integer[], $4::integer[], $5::double precision[]) AS result (country, rank, points, share)
		`, string(scored.Round.ID), pq.Array(countries), pq.Array(ranks), pq.Array(points), pq.Array(shares)); err != nil {
			return fmt.Errorf("failed to keep the round's results: %w", err)
		}
		return nil
	})
	return done, err //nolint:wrapcheck // each step named what failed.
}

func (s *Store) ScoredIn(ctx context.Context, book books.BookID) ([]books.Scored, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT round.round_id, round.book_id, round.times, round.starts_at, scored.ended_at,
		       result.country, result.rank, result.points, result.share
		FROM rounds AS round
		JOIN scored_rounds AS scored USING (round_id)
		LEFT JOIN round_results AS result USING (round_id)
		WHERE round.book_id = $1
		ORDER BY scored.ended_at, round.starts_at, round.round_id, result.rank, result.country
	`, string(book))
	if err != nil {
		return nil, fmt.Errorf("failed to read the rounds the book scored: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var scored []books.Scored
	for rows.Next() {
		var (
			country      sql.NullString
			rank, points sql.NullInt64
			share        sql.NullFloat64
		)
		round, err := scanRound(rows, &country, &rank, &points, &share)
		if err != nil {
			return nil, fmt.Errorf("failed to read a result of a round the book scored: %w", err)
		}
		if len(scored) == 0 || scored[len(scored)-1].Round.ID != round.ID {
			scored = append(scored, books.Scored{Round: round})
		}
		if country.Valid {
			last := &scored[len(scored)-1]
			last.Results = append(last.Results, books.Result{
				Country: books.Country(country.String),
				Rank:    uint32(rank.Int64),   //nolint:gosec // a few hundred countries at most.
				Points:  uint32(points.Int64), //nolint:gosec // 250 a round at most.
				Share:   share.Float64,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the rounds the book scored: %w", err)
	}
	return scored, nil
}

// Count moves each round's mark past the take and counts it in one statement: a take told again is under the mark.
func (s *Store) Count(ctx context.Context, rounds []books.RoundID, take books.Take) error {
	if len(rounds) == 0 {
		return nil
	}
	ids := make([]string, 0, len(rounds))
	for _, round := range rounds {
		ids = append(ids, string(round))
	}
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		WITH counted AS (
			UPDATE rounds SET counted_takes_below = $4
			WHERE round_id = ANY($3) AND counted_takes_below < $4
			RETURNING round_id
		), countries AS (
			INSERT INTO round_takes (round_id, country, tiles)
			SELECT round_id, $2, 1 FROM counted
			ON CONFLICT (round_id, country) DO UPDATE SET tiles = round_takes.tiles + 1
		)
		INSERT INTO round_players (round_id, account_id, country, tiles)
		SELECT round_id, $1, $2, 1 FROM counted
		ON CONFLICT (round_id, account_id, country) DO UPDATE SET tiles = round_players.tiles + 1
	`, uuid.UUID(take.Account), string(take.Country), pq.Array(ids),
		int64(take.Position+1)); err != nil { //nolint:gosec // a ledger position fits in int64.
		return fmt.Errorf("failed to count the take in its rounds: %w", err)
	}
	return nil
}

func (s *Store) ForgetAccount(ctx context.Context, account books.AccountID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM round_players WHERE account_id = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to forget the account's takes in the rounds: %w", err)
	}
	return nil
}

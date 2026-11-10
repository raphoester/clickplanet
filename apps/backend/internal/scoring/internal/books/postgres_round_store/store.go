package postgres_round_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

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
	SELECT round.round_id, round.book_id, round.times, round.ranking, round.starts_at, scored.ended_at
	FROM rounds AS round LEFT JOIN scored_rounds AS scored USING (round_id)
`

type scanner interface {
	Scan(dest ...any) error
}

func scanRound(row scanner, extra ...any) (books.Round, error) {
	var (
		id, book, ranking string
		times             int64
		endedAt           sql.NullTime
		round             books.Round
		err               error
	)
	if err := row.Scan(append([]any{&id, &book, &times, &ranking, &round.StartsAt, &endedAt}, extra...)...); err != nil {
		return books.Round{}, err //nolint:wrapcheck // each caller says what it read.
	}
	if round.Ranking, err = books.RankingNamed(ranking); err != nil {
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

func (s *Store) Open(ctx context.Context, round books.Round, start books.Start) (bool, error) {
	countries := make([]string, 0, len(start))
	tiles := make([]int64, 0, len(start))
	for country, held := range start {
		if held > 0 {
			countries = append(countries, string(country))
			tiles = append(tiles, int64(held))
		}
	}

	done := false
	err := s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		opening, err := tx.ExecContext(ctx, `
			INSERT INTO rounds (round_id, book_id, times, ranking, starts_at) VALUES ($1, $2, $3, $4::round_ranking, $5)
			ON CONFLICT (round_id) DO NOTHING
		`, string(round.ID), string(round.Book), int64(round.Times), round.Ranking.Name(), round.StartsAt)
		if err != nil {
			return fmt.Errorf("failed to open the round: %w", err)
		}
		if done, err = applied(opening, "failed to open the round"); err != nil || !done {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO round_starts (round_id, country, tiles)
			SELECT $1, start.country, start.tiles FROM unnest($2::text[], $3::bigint[]) AS start (country, tiles)
		`, string(round.ID), pq.Array(countries), pq.Array(tiles)); err != nil {
			return fmt.Errorf("failed to keep what each country held when the round opened: %w", err)
		}
		return nil
	})
	return done, err //nolint:wrapcheck // each step named what failed.
}

func (s *Store) StartOf(ctx context.Context, id books.RoundID) (books.Start, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT country, tiles FROM round_starts WHERE round_id = $1`, string(id))
	if err != nil {
		return nil, fmt.Errorf("failed to read what each country held when the round opened: %w", err)
	}
	defer func() { _ = rows.Close() }()

	start := books.Start{}
	for rows.Next() {
		var (
			country string
			tiles   int64
		)
		if err := rows.Scan(&country, &tiles); err != nil {
			return nil, fmt.Errorf("failed to read what a country held when the round opened: %w", err)
		}
		start[books.Country(country)] = uint32(tiles) //nolint:gosec // a country holds a map's tiles at most.
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read what each country held when the round opened: %w", err)
	}
	return start, nil
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
		SELECT round.round_id, round.book_id, round.times, round.ranking, round.starts_at, scored.ended_at,
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

// Count locks the rounds' marks, keeps each round's takes past its mark, and writes what they add up to in three
// statements, whatever the number of takes: each round's new mark, the tiles per flag, and per account and flag.
func (s *Store) Count(ctx context.Context, held map[books.RoundID][]books.Take) error {
	if len(held) == 0 {
		return nil
	}
	ids := make([]string, 0, len(held))
	for round := range held {
		ids = append(ids, string(round))
	}
	slices.Sort(ids)

	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		marks, err := lockMarks(ctx, tx, ids)
		if err != nil {
			return err
		}
		sum := newSum()
		for _, id := range ids {
			if mark, kept := marks[books.RoundID(id)]; kept {
				sum.add(books.RoundID(id), mark, held[books.RoundID(id)])
			}
		}
		if len(sum.rounds) == 0 {
			return nil
		}
		return sum.write(ctx, tx)
	})
}

// lockMarks reads the rounds' marks and holds them until the transaction ends, with the lock an UPDATE of them takes:
// a batch told twice at once is counted by one, then found under the marks by the other.
func lockMarks(ctx context.Context, tx cppg.Querier, ids []string) (map[books.RoundID]books.TakeMark, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT round_id, counted_takes_below, counted_takes_below_index FROM rounds
		WHERE round_id = ANY($1) ORDER BY round_id FOR NO KEY UPDATE
	`, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to lock the rounds' marks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	marks := map[books.RoundID]books.TakeMark{}
	for rows.Next() {
		var (
			id           string
			below, index int64
		)
		if err := rows.Scan(&id, &below, &index); err != nil {
			return nil, fmt.Errorf("failed to read a round's mark: %w", err)
		}
		marks[books.RoundID(id)] = books.TakeMark{Position: uint64(below), Index: uint32(index)} //nolint:gosec // CHECK (>= 0).
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the rounds' marks: %w", err)
	}
	return marks, nil
}

type flagOf struct {
	round   books.RoundID
	country books.Country
}

type playerOf struct {
	round   books.RoundID
	account books.AccountID
	country books.Country
}

// sum is what the takes of a batch add to their rounds. An upsert may take a row once only, so each row is summed here.
type sum struct {
	rounds    []string
	below     []int64
	index     []int64
	countries map[flagOf]int64
	players   map[playerOf]int64
}

func newSum() *sum {
	return &sum{countries: map[flagOf]int64{}, players: map[playerOf]int64{}}
}

func (s *sum) add(round books.RoundID, mark books.TakeMark, takes []books.Take) {
	uncounted, past := mark.Uncounted(takes)
	if len(uncounted) == 0 {
		return
	}
	s.rounds = append(s.rounds, string(round))
	s.below = append(s.below, int64(past.Position)) //nolint:gosec // a ledger position fits in int64.
	s.index = append(s.index, int64(past.Index))
	for _, take := range uncounted {
		s.countries[flagOf{round: round, country: take.Country}]++
		s.players[playerOf{round: round, account: take.Account, country: take.Country}]++
	}
}

func (s *sum) write(ctx context.Context, tx cppg.Querier) error {
	if _, err := tx.ExecContext(ctx, `
		UPDATE rounds SET counted_takes_below = moved.below, counted_takes_below_index = moved.below_index
		FROM unnest($1::text[], $2::bigint[], $3::integer[]) AS moved (round_id, below, below_index)
		WHERE rounds.round_id = moved.round_id
	`, pq.Array(s.rounds), pq.Array(s.below), pq.Array(s.index)); err != nil {
		return fmt.Errorf("failed to move the rounds' marks: %w", err)
	}

	var (
		flagRounds, flags []string
		flagTiles         []int64
	)
	for flag, tiles := range s.countries {
		flagRounds, flags, flagTiles = append(flagRounds, string(flag.round)), append(flags, string(flag.country)), append(flagTiles, tiles)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO round_takes (round_id, country, tiles)
		SELECT * FROM unnest($1::text[], $2::text[], $3::bigint[])
		ON CONFLICT (round_id, country) DO UPDATE SET tiles = round_takes.tiles + excluded.tiles
	`, pq.Array(flagRounds), pq.Array(flags), pq.Array(flagTiles)); err != nil {
		return fmt.Errorf("failed to count the tiles each flag took in the rounds: %w", err)
	}

	var (
		playerRounds, accounts, countries []string
		playerTiles                       []int64
	)
	for player, tiles := range s.players {
		playerRounds, accounts = append(playerRounds, string(player.round)), append(accounts, player.account.String())
		countries, playerTiles = append(countries, string(player.country)), append(playerTiles, tiles)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO round_players (round_id, account_id, country, tiles)
		SELECT * FROM unnest($1::text[], $2::uuid[], $3::text[], $4::bigint[])
		ON CONFLICT (round_id, account_id, country) DO UPDATE SET tiles = round_players.tiles + excluded.tiles
	`, pq.Array(playerRounds), pq.Array(accounts), pq.Array(countries), pq.Array(playerTiles)); err != nil {
		return fmt.Errorf("failed to count the tiles each account took in the rounds: %w", err)
	}
	return nil
}

func (s *Store) ForgetAccount(ctx context.Context, account books.AccountID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM round_players WHERE account_id = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to forget the account's takes in the rounds: %w", err)
	}
	return nil
}

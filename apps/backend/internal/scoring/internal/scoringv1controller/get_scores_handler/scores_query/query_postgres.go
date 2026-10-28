package scores_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var ErrNoBook = errors.New("no book has this id")

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

func (q *PostgresQuery) Scores(ctx context.Context, book books.BookID) (*scoringv1.Scores, error) {
	ledger, err := q.ledger(ctx, book)
	if err != nil {
		return nil, err
	}
	return &scoringv1.Scores{BookId: string(book), Scores: scoresOf(ledger.table(ledger.rounds))}, nil
}

// Live is the scores a stream shows: there are always some.
func (q *PostgresQuery) Live(ctx context.Context, book books.BookID) (*scoringv1.Scores, bool, error) {
	scores, err := q.Scores(ctx, book)
	return scores, err == nil, err
}

// Around is the book's scores before a round that ended at a time, and after it.
func (q *PostgresQuery) Around(ctx context.Context, book books.BookID, endedAt time.Time) ([]*scoringv1.Score, []*scoringv1.Score, error) {
	ledger, err := q.ledger(ctx, book)
	if err != nil {
		return nil, nil, err
	}
	var before, after []books.Scored
	for _, round := range ledger.rounds {
		if round.Round.EndsAt.Before(endedAt) {
			before = append(before, round)
		}
		if !round.Round.EndsAt.After(endedAt) {
			after = append(after, round)
		}
	}
	return scoresOf(ledger.table(before)), scoresOf(ledger.table(after)), nil
}

func (q *PostgresQuery) Table(ctx context.Context, book books.BookID) (*scoringv1.GetTableResponse, error) {
	ledger, err := q.ledger(ctx, book)
	if err != nil {
		return nil, err
	}
	if !ledger.found {
		return nil, fmt.Errorf("%w: %q", ErrNoBook, book)
	}
	table := ledger.table(ledger.rounds)
	places := make([]*scoringv1.TablePlace, 0, len(table))
	for _, placed := range table {
		places = append(places, &scoringv1.TablePlace{
			CountryId:      string(placed.Country),
			Rank:           placed.Rank,
			Points:         uint32(placed.Points), //nolint:gosec // 25 a day for a few weeks.
			RoundsWon:      placed.RoundsWon,
			TiebreakPoints: uint32(placed.TiebreakPoints), //nolint:gosec // 250 a round at most.
		})
	}
	return &scoringv1.GetTableResponse{Table: places, Closed: ledger.book.Closed}, nil
}

type ledger struct {
	found  bool
	book   books.Book
	rounds []books.Scored
}

func (l ledger) table(rounds []books.Scored) []books.Placed {
	return l.book.Table(rounds)
}

func scoresOf(table []books.Placed) []*scoringv1.Score {
	shown := make([]*scoringv1.Score, 0, len(table))
	for _, placed := range table {
		shown = append(shown, &scoringv1.Score{
			Rank:      placed.Rank,
			CountryId: string(placed.Country),
			Points:    uint32(placed.Points), //nolint:gosec // 25 a day for a few weeks.
			RoundsWon: placed.RoundsWon,
		})
	}
	return shown
}

func (q *PostgresQuery) ledger(ctx context.Context, id books.BookID) (ledger, error) {
	var (
		kept     = ledger{book: books.Book{ID: id}}
		endedAt  sql.NullTime
		tiebreak sql.NullString
	)
	err := q.db.On(ctx).QueryRowContext(ctx, `
		SELECT book.starts_at, closed.ended_at, tiebreak.round_id
		FROM books AS book
		LEFT JOIN closed_books AS closed USING (book_id)
		LEFT JOIN book_tiebreaks AS tiebreak USING (book_id)
		WHERE book.book_id = $1
	`, string(id)).Scan(&kept.book.StartsAt, &endedAt, &tiebreak)
	if errors.Is(err, sql.ErrNoRows) {
		return kept, nil
	}
	if err != nil {
		return ledger{}, fmt.Errorf("failed to read the book: %w", err)
	}
	kept.found = true
	if endedAt.Valid {
		kept.book = kept.book.ClosedAt(endedAt.Time, books.RoundID(tiebreak.String))
	}

	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT round.round_id, scored.ended_at, result.country, result.rank, result.points
		FROM rounds AS round
		JOIN scored_rounds AS scored USING (round_id)
		JOIN round_results AS result USING (round_id)
		WHERE round.book_id = $1 AND result.points > 0
		ORDER BY scored.ended_at, round.round_id
	`, string(id))
	if err != nil {
		return ledger{}, fmt.Errorf("failed to read the book's points: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			round        string
			ended        time.Time
			country      string
			rank, points int64
		)
		if err := rows.Scan(&round, &ended, &country, &rank, &points); err != nil {
			return ledger{}, fmt.Errorf("failed to read a country's points: %w", err)
		}
		if len(kept.rounds) == 0 || string(kept.rounds[len(kept.rounds)-1].Round.ID) != round {
			kept.rounds = append(kept.rounds, books.Scored{Round: books.Round{ID: books.RoundID(round), Book: id, Scored: true, EndsAt: ended}})
		}
		last := &kept.rounds[len(kept.rounds)-1]
		last.Results = append(last.Results, books.Result{
			Country: books.Country(country),
			Rank:    uint32(rank),   //nolint:gosec // a few hundred countries at most.
			Points:  uint32(points), //nolint:gosec // 250 a round at most.
		})
	}
	if err := rows.Err(); err != nil {
		return ledger{}, fmt.Errorf("failed to read the book's points: %w", err)
	}
	return kept, nil
}

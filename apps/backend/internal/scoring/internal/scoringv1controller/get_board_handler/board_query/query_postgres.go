package board_query

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

type Authors interface {
	Authors(ctx context.Context, accounts []standings.AccountID) (map[standings.AccountID]*playerv1.Author, error)
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

var ErrUnknownCountry = errors.New("not a country")

const (
	Shown = 10
	page  = 200
)

func NewPostgresQuery(db cppg.Conn, authors Authors, countries CountryChecker) *PostgresQuery {
	return &PostgresQuery{db: db, authors: authors, countries: countries}
}

type PostgresQuery struct {
	db        cppg.Conn
	authors   Authors
	countries CountryChecker
}

// View is a book's board, of every flag or of one.
type View struct {
	Book    books.BookID
	Country string
}

func (q *PostgresQuery) Board(ctx context.Context, view View) (*scoringv1.GetBoardResponse, error) {
	top, err := q.top(ctx, view)
	if err != nil {
		return nil, err
	}
	return &scoringv1.GetBoardResponse{Standings: top}, nil
}

// Live is the board a stream shows: there is always one.
func (q *PostgresQuery) Live(ctx context.Context, view View) (*scoringv1.Board, bool, error) {
	top, err := q.top(ctx, view)
	if err != nil {
		return nil, false, err
	}
	return &scoringv1.Board{BookId: string(view.Book), Standings: top}, true, nil
}

func (q *PostgresQuery) top(ctx context.Context, view View) ([]*scoringv1.Standing, error) {
	country := view.Country
	if country != "" && !q.countries.CheckCountry(country) {
		return nil, fmt.Errorf("%w: %q", ErrUnknownCountry, country)
	}

	top := make([]*scoringv1.Standing, 0, Shown)
	after := start
	for {
		lines, err := q.lines(ctx, view.Book, country, after)
		if err != nil {
			return nil, err
		}
		named, err := q.named(ctx, lines)
		if err != nil {
			return nil, err
		}
		for _, line := range lines {
			author, known := named[line.account]
			if !known || author.GetGuest() {
				continue
			}
			top = append(top, &scoringv1.Standing{
				Rank:      rankAfter(top, line.tiles),
				AccountId: line.account.String(),
				Name:      author.GetName(),
				Color:     author.GetColor(),
				CountryId: line.country,
				Tiles:     line.tiles,
				WornTitle: author.GetWornTitle(),
			})
			if len(top) == Shown {
				return top, nil
			}
		}
		if len(lines) < page {
			return top, nil
		}
		after = lines[len(lines)-1]
	}
}

type line struct {
	account standings.AccountID
	country string
	tiles   uint64
}

var start = line{tiles: math.MaxInt64}

const mainLines = `
	SELECT account_id, country, tiles
	FROM contributions
	WHERE book_id = $1 AND main AND (tiles < $2 OR (tiles = $2 AND account_id > $3))
	ORDER BY tiles DESC, account_id
	LIMIT $4
`

const linesOfCountry = `
	SELECT account_id, country, tiles
	FROM contributions
	WHERE book_id = $1 AND country = $5 AND (tiles < $2 OR (tiles = $2 AND account_id > $3))
	ORDER BY tiles DESC, account_id
	LIMIT $4
`

func (q *PostgresQuery) lines(ctx context.Context, book books.BookID, country string, after line) ([]line, error) {
	query, args := mainLines, []any{string(book), int64(after.tiles), uuid.UUID(after.account), page} //nolint:gosec // a line's tiles fit in int64.
	if country != "" {
		query, args = linesOfCountry, append(args, country)
	}

	rows, err := q.db.On(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to read the standings: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var lines []line
	for rows.Next() {
		var (
			account uuid.UUID
			each    line
			tiles   int64
		)
		if err := rows.Scan(&account, &each.country, &tiles); err != nil {
			return nil, fmt.Errorf("failed to scan a line of the standings: %w", err)
		}
		each.account = standings.AccountID(account)
		each.tiles = uint64(tiles) //nolint:gosec // CHECK (tiles > 0).
		lines = append(lines, each)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the standings: %w", err)
	}
	return lines, nil
}

func (q *PostgresQuery) named(ctx context.Context, lines []line) (map[standings.AccountID]*playerv1.Author, error) {
	if len(lines) == 0 {
		return map[standings.AccountID]*playerv1.Author{}, nil
	}
	accounts := make([]standings.AccountID, len(lines))
	for i, line := range lines {
		accounts[i] = line.account
	}
	named, err := q.authors.Authors(ctx, accounts)
	if err != nil {
		return nil, fmt.Errorf("failed to read who the standings are: %w", err)
	}
	return named, nil
}

func rankAfter(top []*scoringv1.Standing, tiles uint64) uint32 {
	if len(top) > 0 && top[len(top)-1].GetTiles() == tiles {
		return top[len(top)-1].GetRank()
	}
	return uint32(len(top) + 1) //nolint:gosec // at most Shown.
}

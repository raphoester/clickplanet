package board_query

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/scoringread"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

type CountryChecker interface {
	CheckCountry(country string) bool
}

var ErrUnknownCountry = errors.New("not a country")

const Shown = 10

func NewPostgresQuery(db cppg.Conn, countries CountryChecker) *PostgresQuery {
	return &PostgresQuery{db: db, countries: countries}
}

type PostgresQuery struct {
	db        cppg.Conn
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

	query, args := mainLines, []any{string(view.Book), Shown}
	if country != "" {
		query, args = linesOfCountry, append(args, country)
	}
	rows, err := q.db.On(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to read the standings: %w", err)
	}
	defer func() { _ = rows.Close() }()

	top := make([]*scoringv1.Standing, 0, Shown)
	for rows.Next() {
		var (
			account uuid.UUID
			each    = &scoringv1.Standing{}
			tiles   int64
		)
		if err := rows.Scan(&account, &each.CountryId, &tiles); err != nil {
			return nil, fmt.Errorf("failed to scan a line of the standings: %w", err)
		}
		each.AccountId = standings.AccountID(account).String()
		each.Tiles = uint64(tiles) //nolint:gosec // CHECK (tiles > 0).
		each.Rank = rankAfter(top, each.GetTiles())
		top = append(top, each)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the standings: %w", err)
	}
	return top, nil
}

var (
	mainLines = `
		SELECT account_id, country, tiles
		FROM contributions
		WHERE book_id = $1 AND main AND ` + scoringread.Ranked("contributions.account_id") + `
		ORDER BY tiles DESC, account_id
		LIMIT $2
	`
	linesOfCountry = `
		SELECT account_id, country, tiles
		FROM contributions
		WHERE book_id = $1 AND country = $3 AND ` + scoringread.Ranked("contributions.account_id") + `
		ORDER BY tiles DESC, account_id
		LIMIT $2
	`
)

func rankAfter(top []*scoringv1.Standing, tiles uint64) uint32 {
	if len(top) > 0 && top[len(top)-1].GetTiles() == tiles {
		return top[len(top)-1].GetRank()
	}
	return uint32(len(top) + 1) //nolint:gosec // at most Shown.
}

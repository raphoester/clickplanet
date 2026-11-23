package my_tally_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

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

func NewPostgresQuery(db cppg.Conn, countries CountryChecker) *PostgresQuery {
	return &PostgresQuery{db: db, countries: countries}
}

type PostgresQuery struct {
	db        cppg.Conn
	countries CountryChecker
}

// A guest, or an account asked deleted, has its tiles and no rank: no board ranks it.
func (q *PostgresQuery) MyTally(
	ctx context.Context,
	book books.BookID,
	account standings.AccountID,
	country string,
) (*scoringv1.GetMyTallyResponse, error) {
	if country != "" && !q.countries.CheckCountry(country) {
		return nil, fmt.Errorf("%w: %q", ErrUnknownCountry, country)
	}

	mine, ranked, err := q.tiles(ctx, book, account, country)
	if err != nil {
		return nil, err
	}
	if mine.GetTiles() == 0 || !ranked {
		return mine, nil
	}

	var global, inCountry uint32
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() (err error) {
		global, err = q.rank(ctx, aboveOnTheMap, string(book), int64(mine.GetTiles())) //nolint:gosec // a line's tiles fit in int64.
		return err
	})
	if mine.GetCountryTiles() > 0 {
		group.Go(func() (err error) {
			inCountry, err = q.rank(ctx, aboveInTheCountry, string(book), int64(mine.GetCountryTiles()), country) //nolint:gosec // a line's tiles fit in int64.
			return err
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err //nolint:wrapcheck // each part already names what failed.
	}
	mine.GlobalRank, mine.CountryRank = global, inCountry
	return mine, nil
}

var tilesOfMine = `
	SELECT country, tiles, COALESCE((
		SELECT tiles FROM contributions WHERE book_id = $1 AND account_id = $2 AND country = $3
	), 0), ` + scoringread.Ranked("contributions.account_id") + `
	FROM contributions
	WHERE book_id = $1 AND account_id = $2 AND main
`

func (q *PostgresQuery) tiles(
	ctx context.Context,
	book books.BookID,
	account standings.AccountID,
	country string,
) (*scoringv1.GetMyTallyResponse, bool, error) {
	var (
		mine                  = &scoringv1.GetMyTallyResponse{}
		tiles, tilesOfCountry int64
		ranked                bool
	)
	err := q.db.On(ctx).QueryRowContext(ctx, tilesOfMine, string(book), uuid.UUID(account), country).
		Scan(&mine.CountryId, &tiles, &tilesOfCountry, &ranked)
	if errors.Is(err, sql.ErrNoRows) {
		return mine, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("failed to read the caller's tally: %w", err)
	}
	mine.Tiles = uint64(tiles)                 //nolint:gosec // CHECK (tiles > 0).
	mine.CountryTiles = uint64(tilesOfCountry) //nolint:gosec // CHECK (tiles > 0), or 0.
	return mine, ranked, nil
}

var (
	aboveOnTheMap = `
		SELECT count(*)
		FROM contributions
		WHERE book_id = $1 AND main AND tiles > $2 AND ` + scoringread.Ranked("contributions.account_id") + `
	`
	aboveInTheCountry = `
		SELECT count(*)
		FROM contributions
		WHERE book_id = $1 AND tiles > $2 AND country = $3 AND ` + scoringread.Ranked("contributions.account_id") + `
	`
)

func (q *PostgresQuery) rank(ctx context.Context, above string, args ...any) (uint32, error) {
	var ahead uint32
	if err := q.db.On(ctx).QueryRowContext(ctx, above, args...).Scan(&ahead); err != nil {
		return 0, fmt.Errorf("failed to count who is above the caller: %w", err)
	}
	return ahead + 1, nil
}

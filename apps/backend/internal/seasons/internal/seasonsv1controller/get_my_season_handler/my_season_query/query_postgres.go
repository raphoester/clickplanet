package my_season_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Authors interface {
	Authors(ctx context.Context, accounts []standings.AccountID) (map[standings.AccountID]*playerv1.Author, error)
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

var ErrUnknownCountry = errors.New("not a country")

const page = 500

func NewPostgresQuery(
	db cppg.Querier,
	authors Authors,
	seasons calendar.Calendar,
	clock cptime.Clock,
	countries CountryChecker,
) *PostgresQuery {
	return &PostgresQuery{db: db, authors: authors, seasons: seasons, clock: clock, countries: countries}
}

type PostgresQuery struct {
	db        cppg.Querier
	authors   Authors
	seasons   calendar.Calendar
	clock     cptime.Clock
	countries CountryChecker
}

func (q *PostgresQuery) MySeason(
	ctx context.Context,
	account standings.AccountID,
	country string,
) (*seasonsv1.GetMySeasonResponse, error) {
	if country != "" && !q.countries.CheckCountry(country) {
		return nil, fmt.Errorf("%w: %q", ErrUnknownCountry, country)
	}
	season, ok := q.seasons.Current(q.clock.Now())
	if !ok {
		return &seasonsv1.GetMySeasonResponse{}, nil
	}

	mine, err := q.tiles(ctx, season.Number, account, country)
	if err != nil {
		return nil, err
	}
	if mine.GetTiles() == 0 {
		return mine, nil
	}

	named, err := q.authors.Authors(ctx, []standings.AccountID{account})
	if err != nil {
		return nil, fmt.Errorf("failed to read who the caller is: %w", err)
	}
	if !ranked(named, account) {
		return mine, nil
	}
	mine.WornTitle = named[account].GetWornTitle()

	var global, inCountry uint32
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() (err error) {
		global, err = q.rank(ctx, aboveOnTheMap, int64(season.Number), int64(mine.GetTiles())) //nolint:gosec // a line's tiles fit in int64.
		return err
	})
	if mine.GetCountryTiles() > 0 {
		group.Go(func() (err error) {
			inCountry, err = q.rank(ctx, aboveInTheCountry, int64(season.Number), int64(mine.GetCountryTiles()), country) //nolint:gosec // a line's tiles fit in int64.
			return err
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err //nolint:wrapcheck // each part already names what failed.
	}
	mine.GlobalRank, mine.CountryRank = global, inCountry
	return mine, nil
}

const tilesOfMine = `
	SELECT country, tiles, COALESCE((
		SELECT tiles FROM contributions WHERE season = $1 AND account_id = $2 AND country = $3
	), 0)
	FROM contributions
	WHERE season = $1 AND account_id = $2 AND main
`

func (q *PostgresQuery) tiles(
	ctx context.Context,
	season calendar.Number,
	account standings.AccountID,
	country string,
) (*seasonsv1.GetMySeasonResponse, error) {
	var (
		mine                  = &seasonsv1.GetMySeasonResponse{}
		tiles, tilesOfCountry int64
	)
	err := q.db.QueryRowContext(ctx, tilesOfMine, int64(season), uuid.UUID(account), country).
		Scan(&mine.CountryId, &tiles, &tilesOfCountry)
	if errors.Is(err, sql.ErrNoRows) {
		return mine, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the caller's season: %w", err)
	}
	mine.Tiles = uint64(tiles)                 //nolint:gosec // CHECK (tiles > 0).
	mine.CountryTiles = uint64(tilesOfCountry) //nolint:gosec // CHECK (tiles > 0), or 0.
	return mine, nil
}

const aboveOnTheMap = `
	SELECT account_id
	FROM contributions
	WHERE season = $1 AND main AND tiles > $2 AND account_id > $3
	ORDER BY account_id
	LIMIT $4
`

const aboveInTheCountry = `
	SELECT account_id
	FROM contributions
	WHERE season = $1 AND tiles > $2 AND country = $3 AND account_id > $4
	ORDER BY account_id
	LIMIT $5
`

func (q *PostgresQuery) rank(ctx context.Context, above string, args ...any) (uint32, error) {
	rank := uint32(1)
	var after standings.AccountID
	for {
		accounts, err := q.above(ctx, above, slices.Concat(args, []any{uuid.UUID(after), page}))
		if err != nil {
			return 0, err
		}
		ahead, err := q.ahead(ctx, accounts)
		if err != nil {
			return 0, err
		}
		rank += ahead
		if len(accounts) < page {
			return rank, nil
		}
		after = accounts[len(accounts)-1]
	}
}

func (q *PostgresQuery) above(ctx context.Context, query string, args []any) ([]standings.AccountID, error) {
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to read who is above the caller: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var above []standings.AccountID
	for rows.Next() {
		var account uuid.UUID
		if err := rows.Scan(&account); err != nil {
			return nil, fmt.Errorf("failed to scan who is above the caller: %w", err)
		}
		above = append(above, standings.AccountID(account))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read who is above the caller: %w", err)
	}
	return above, nil
}

func (q *PostgresQuery) ahead(ctx context.Context, accounts []standings.AccountID) (uint32, error) {
	if len(accounts) == 0 {
		return 0, nil
	}
	named, err := q.authors.Authors(ctx, accounts)
	if err != nil {
		return 0, fmt.Errorf("failed to read who is above the caller: %w", err)
	}
	var count uint32
	for _, account := range accounts {
		if ranked(named, account) {
			count++
		}
	}
	return count, nil
}

func ranked(named map[standings.AccountID]*playerv1.Author, account standings.AccountID) bool {
	author, known := named[account]
	return known && !author.GetGuest()
}

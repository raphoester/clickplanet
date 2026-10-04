package my_season_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

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

const page = 500

func NewPostgresQuery(db cppg.Querier, authors Authors, seasons calendar.Calendar, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, authors: authors, seasons: seasons, clock: clock}
}

type PostgresQuery struct {
	db      cppg.Querier
	authors Authors
	seasons calendar.Calendar
	clock   cptime.Clock
}

func (q *PostgresQuery) MySeason(ctx context.Context, account standings.AccountID) (*seasonsv1.GetMySeasonResponse, error) {
	season, ok := q.seasons.Current(q.clock.Now())
	if !ok {
		return &seasonsv1.GetMySeasonResponse{}, nil
	}

	mine := &seasonsv1.GetMySeasonResponse{}
	var tiles int64
	err := q.db.QueryRowContext(ctx, `
		SELECT country, tiles FROM contributions WHERE season = $1 AND account_id = $2 AND main
	`, int64(season.Number), uuid.UUID(account)).Scan(&mine.CountryId, &tiles)
	if errors.Is(err, sql.ErrNoRows) {
		return mine, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the caller's season: %w", err)
	}
	mine.Tiles = uint64(tiles) //nolint:gosec // CHECK (tiles > 0).

	named, err := q.authors.Authors(ctx, []standings.AccountID{account})
	if err != nil {
		return nil, fmt.Errorf("failed to read who the caller is: %w", err)
	}
	if !ranked(named, account) {
		return mine, nil
	}

	mine.GlobalRank, mine.CountryRank = 1, 1
	var after standings.AccountID
	for {
		above, err := q.above(ctx, season.Number, mine, after)
		if err != nil {
			return nil, err
		}
		if len(above) == 0 {
			return mine, nil
		}
		if err := q.count(ctx, above, mine); err != nil {
			return nil, err
		}
		if len(above) < page {
			return mine, nil
		}
		after = above[len(above)-1].account
	}
}

func (q *PostgresQuery) count(ctx context.Context, above []aboveLine, mine *seasonsv1.GetMySeasonResponse) error {
	accounts := make([]standings.AccountID, len(above))
	for i, other := range above {
		accounts[i] = other.account
	}
	named, err := q.authors.Authors(ctx, accounts)
	if err != nil {
		return fmt.Errorf("failed to read who is above the caller: %w", err)
	}
	for _, other := range above {
		if !ranked(named, other.account) {
			continue
		}
		mine.GlobalRank++
		if other.sameCountry {
			mine.CountryRank++
		}
	}
	return nil
}

type aboveLine struct {
	account     standings.AccountID
	sameCountry bool
}

const aboveMine = `
	SELECT account_id, country = $3
	FROM contributions
	WHERE season = $1 AND main AND tiles > $2 AND account_id > $4
	ORDER BY account_id
	LIMIT $5
`

func (q *PostgresQuery) above(
	ctx context.Context,
	season calendar.Number,
	mine *seasonsv1.GetMySeasonResponse,
	after standings.AccountID,
) ([]aboveLine, error) {
	rows, err := q.db.QueryContext(ctx, aboveMine,
		int64(season), int64(mine.GetTiles()), mine.GetCountryId(), uuid.UUID(after), page) //nolint:gosec // a line's tiles fit in int64.
	if err != nil {
		return nil, fmt.Errorf("failed to read who is above the caller: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var above []aboveLine
	for rows.Next() {
		var (
			account uuid.UUID
			each    aboveLine
		)
		if err := rows.Scan(&account, &each.sameCountry); err != nil {
			return nil, fmt.Errorf("failed to scan who is above the caller: %w", err)
		}
		each.account = standings.AccountID(account)
		above = append(above, each)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read who is above the caller: %w", err)
	}
	return above, nil
}

func ranked(named map[standings.AccountID]*playerv1.Author, account standings.AccountID) bool {
	author, known := named[account]
	return known && !author.GetGuest()
}

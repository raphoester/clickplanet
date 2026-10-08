package playerread

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const fronts = `
	SELECT plays_for, country, tiles FROM (
		SELECT true AS plays_for, country, plays_for AS tiles, row_number() OVER (ORDER BY plays_for DESC, country) AS place
		FROM fronts WHERE account_id = $1 AND plays_for > 0
		UNION ALL
		SELECT false, country, plays_against, row_number() OVER (ORDER BY plays_against DESC, country)
		FROM fronts WHERE account_id = $1 AND plays_against > 0
	) ranked
	WHERE $2::bigint IS NULL OR place <= $2
	ORDER BY tiles DESC, country
`

func Fronts(ctx context.Context, db cppg.Querier, account uuid.UUID) (playsFor, playsAgainst []*playerv1.CountryTiles, err error) {
	return frontsUpTo(ctx, db, account, sql.NullInt64{})
}

func TopFronts(ctx context.Context, db cppg.Querier, account uuid.UUID, most int64) (playsFor, playsAgainst []*playerv1.CountryTiles, err error) {
	return frontsUpTo(ctx, db, account, sql.NullInt64{Int64: most, Valid: true})
}

func frontsUpTo(ctx context.Context, db cppg.Querier, account uuid.UUID, most sql.NullInt64) (playsFor, playsAgainst []*playerv1.CountryTiles, err error) {
	rows, err := db.QueryContext(ctx, fronts, account, most)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read the countries the player plays for and against: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			isFor   bool
			country = &playerv1.CountryTiles{}
		)
		if err := rows.Scan(&isFor, &country.CountryId, &country.Tiles); err != nil {
			return nil, nil, fmt.Errorf("failed to read a country the player plays for or against: %w", err)
		}
		if isFor {
			playsFor = append(playsFor, country)
		} else {
			playsAgainst = append(playsAgainst, country)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("failed to read the countries the player plays for and against: %w", err)
	}
	return playsFor, playsAgainst, nil
}

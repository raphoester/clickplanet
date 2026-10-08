package race_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func NewPostgresQuery(db cppg.Querier, seasons calendar.Calendar, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, seasons: seasons, clock: clock}
}

type PostgresQuery struct {
	db      cppg.Querier
	seasons calendar.Calendar
	clock   cptime.Clock
}

func (q *PostgresQuery) Race(ctx context.Context) (*seasonsv1.Race, error) {
	now := q.clock.Now()
	race := &seasonsv1.Race{}
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() (err error) {
		race.Closed, err = q.closed(ctx)
		return err
	})
	if round, ok := rounds.Current(q.seasons, now); ok {
		group.Go(func() (err error) {
			race.Round, err = q.round(ctx, round)
			return err
		})
		group.Go(func() (err error) {
			race.Scores, err = q.scores(ctx, round.Season, through, now)
			return err
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err //nolint:wrapcheck // each part named what failed.
	}
	return race, nil
}

func (q *PostgresQuery) closed(ctx context.Context) (*seasonsv1.ClosedRound, error) {
	var (
		season, earlier, samples, mapTiles int64
		round                              rounds.Round
	)
	err := q.db.QueryRowContext(ctx, `
		SELECT round.season, round.ends_at, round.finale, round.samples, round.map_tiles,
		       (SELECT count(*) FROM rounds AS earlier WHERE earlier.season = round.season AND earlier.ends_at < round.ends_at)
		FROM rounds AS round WHERE round.closed ORDER BY round.ends_at DESC, round.season DESC LIMIT 1
	`).Scan(&season, &round.EndsAt, &round.Finale, &samples, &mapTiles, &earlier)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil //nolint:nilnil // no round closed yet: the race carries none.
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the round closed last: %w", err)
	}
	round.Season = calendar.Number(season) //nolint:gosec // CHECK (season >= 0).

	closed := &seasonsv1.ClosedRound{
		Season:        uint32(round.Season),
		Number:        uint32(earlier + 1), //nolint:gosec // a round a day.
		EndedAtUnixMs: round.EndsAt.UnixMilli(),
		Finale:        round.Finale,
	}
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() (err error) {
		closed.Standings, err = q.results(ctx, round, samples*mapTiles)
		return err
	})
	group.Go(func() (err error) {
		closed.Before, err = q.scores(ctx, round.Season, before, round.EndsAt)
		return err
	})
	group.Go(func() (err error) {
		closed.After, err = q.scores(ctx, round.Season, through, round.EndsAt)
		return err
	})
	if err := group.Wait(); err != nil {
		return nil, err //nolint:wrapcheck // each part named what failed.
	}
	return closed, nil
}

func (q *PostgresQuery) results(ctx context.Context, round rounds.Round, counted int64) ([]*seasonsv1.RoundStanding, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT result.rank, result.country, result.points, coalesce(held.tiles, 0)
		FROM round_results AS result LEFT JOIN round_holdings AS held USING (season, ends_at, country)
		WHERE result.season = $1 AND result.ends_at = $2 AND result.points > 0
		ORDER BY result.rank, result.country
	`, int64(round.Season), round.EndsAt)
	if err != nil {
		return nil, fmt.Errorf("failed to read the results of the round closed last: %w", err)
	}
	defer func() { _ = rows.Close() }()

	standings := []*seasonsv1.RoundStanding{}
	for rows.Next() {
		var (
			rank, points int64
			country      string
			tiles        int64
		)
		if err := rows.Scan(&rank, &country, &points, &tiles); err != nil {
			return nil, fmt.Errorf("failed to read a result of the round closed last: %w", err)
		}
		standing := &seasonsv1.RoundStanding{
			Rank:      uint32(rank), //nolint:gosec // a few hundred countries at most.
			CountryId: country,
			Points:    uint32(points), //nolint:gosec // 75 a round at most.
		}
		if counted > 0 {
			standing.Share = float64(tiles) / float64(counted)
		}
		standings = append(standings, standing)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the results of the round closed last: %w", err)
	}
	return standings, nil
}

func (q *PostgresQuery) round(ctx context.Context, round rounds.Round) (*seasonsv1.Round, error) {
	var (
		earlier           int64
		samples, mapTiles sql.NullInt64
	)
	if err := q.db.QueryRowContext(ctx, `
		SELECT count(*) FILTER (WHERE ends_at < $2), max(samples) FILTER (WHERE ends_at = $2), max(map_tiles) FILTER (WHERE ends_at = $2)
		FROM rounds WHERE season = $1
	`, int64(round.Season), round.EndsAt).Scan(&earlier, &samples, &mapTiles); err != nil {
		return nil, fmt.Errorf("failed to read the round in progress: %w", err)
	}

	shown := &seasonsv1.Round{
		Number:       uint32(earlier + 1), //nolint:gosec // a round a day.
		EndsAtUnixMs: round.EndsAt.UnixMilli(),
		Finale:       round.Finale,
		Standings:    []*seasonsv1.RoundStanding{},
	}
	if !samples.Valid {
		return shown, nil
	}

	held, err := q.held(ctx, round)
	if err != nil {
		return nil, err
	}
	for _, result := range round.Results(held) {
		shown.Standings = append(shown.Standings, &seasonsv1.RoundStanding{
			Rank:      result.Rank,
			CountryId: string(result.Country),
			Share:     float64(held[result.Country]) / float64(samples.Int64) / float64(mapTiles.Int64),
			Points:    result.Points,
		})
	}
	return shown, nil
}

func (q *PostgresQuery) held(ctx context.Context, round rounds.Round) (map[rounds.Country]uint64, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT country, tiles FROM round_holdings WHERE season = $1 AND ends_at = $2
	`, int64(round.Season), round.EndsAt)
	if err != nil {
		return nil, fmt.Errorf("failed to read what each country holds in the round: %w", err)
	}
	defer func() { _ = rows.Close() }()

	held := map[rounds.Country]uint64{}
	for rows.Next() {
		var (
			country string
			tiles   int64
		)
		if err := rows.Scan(&country, &tiles); err != nil {
			return nil, fmt.Errorf("failed to read what a country holds in the round: %w", err)
		}
		held[rounds.Country(country)] = uint64(tiles) //nolint:gosec // CHECK (tiles > 0).
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read what each country holds in the round: %w", err)
	}
	return held, nil
}

type bound string

const (
	before  bound = "<"
	through bound = "<="
)

func (q *PostgresQuery) scores(ctx context.Context, season calendar.Number, bound bound, at time.Time) ([]*seasonsv1.Score, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT result.country,
		       sum(result.points),
		       count(*) FILTER (WHERE result.rank = 1),
		       coalesce(sum(result.points) FILTER (WHERE round.finale), 0)
		FROM round_results AS result JOIN rounds AS round USING (season, ends_at)
		WHERE result.season = $1 AND result.ends_at `+string(bound)+` $2
		GROUP BY result.country
		HAVING sum(result.points) > 0
	`, int64(season), at)
	if err != nil {
		return nil, fmt.Errorf("failed to read the season's points: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var scores []rounds.Score
	for rows.Next() {
		var (
			country                 string
			points, won, finalePart int64
		)
		if err := rows.Scan(&country, &points, &won, &finalePart); err != nil {
			return nil, fmt.Errorf("failed to read a country's points: %w", err)
		}
		scores = append(scores, rounds.Score{
			Country:      rounds.Country(country),
			Points:       uint64(points),     //nolint:gosec // CHECK (points >= 0).
			RoundsWon:    uint32(won),        //nolint:gosec // a round a day.
			FinalePoints: uint64(finalePart), //nolint:gosec // CHECK (points >= 0).
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the season's points: %w", err)
	}

	table := rounds.TableOf(scores)
	shown := make([]*seasonsv1.Score, 0, len(table))
	for _, placed := range table {
		shown = append(shown, &seasonsv1.Score{
			Rank:      placed.Rank,
			CountryId: string(placed.Country),
			Points:    uint32(placed.Points), //nolint:gosec // 75 a round at most.
			RoundsWon: placed.RoundsWon,
		})
	}
	return shown, nil
}

package race_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Territory interface {
	Snapshot(ctx context.Context) (rounds.Snapshot, error)
}

type Authors interface {
	Authors(ctx context.Context, accounts []rounds.AccountID) (map[rounds.AccountID]*playerv1.Author, error)
}

const (
	// Shown is how many players a round of its own names.
	Shown = 10
	// read is how many of its best players are asked for, to name Shown who are not guests.
	read = 50
)

func NewPostgresQuery(
	db cppg.Conn,
	territory Territory,
	authors Authors,
	seasons calendar.Calendar,
	clock cptime.Clock,
) *PostgresQuery {
	return &PostgresQuery{db: db, territory: territory, authors: authors, seasons: seasons, clock: clock}
}

type PostgresQuery struct {
	db        cppg.Conn
	territory Territory
	authors   Authors
	seasons   calendar.Calendar
	clock     cptime.Clock
}

func (q *PostgresQuery) Race(ctx context.Context) (*seasonsv1.Race, error) {
	now := q.clock.Now()
	race := &seasonsv1.Race{}
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() (err error) {
		race.Closed, err = q.closed(ctx)
		return err
	})
	if season, ok := q.seasons.Current(now); ok {
		group.Go(func() (err error) {
			race.Round, err = q.round(ctx, season, now)
			return err
		})
		group.Go(func() (err error) {
			race.Scores, err = q.scores(ctx, season.Number, through, now)
			return err
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err //nolint:wrapcheck // each part named what failed.
	}
	return race, nil
}

const roundColumns = `round.season, round.ends_at, round.own, round.finale, round.starts_at, round.times,
	(SELECT count(*) FROM rounds AS earlier
	 WHERE earlier.season = round.season AND earlier.own = round.own AND earlier.ends_at < round.ends_at)`

type scanner interface {
	Scan(dest ...any) error
}

func scanRound(row scanner) (rounds.Round, uint32, error) {
	var (
		season, times, earlier int64
		startsAt               sql.NullTime
		round                  rounds.Round
	)
	if err := row.Scan(&season, &round.EndsAt, &round.Own, &round.Finale, &startsAt, &times, &earlier); err != nil {
		return rounds.Round{}, 0, err //nolint:wrapcheck // each caller says what it read.
	}
	round.Season = calendar.Number(season) //nolint:gosec // CHECK (season >= 0).
	round.EndsAt = round.EndsAt.UTC()
	round.Times = uint32(times) //nolint:gosec // CHECK (times BETWEEN 1 AND 10).
	if startsAt.Valid {
		round.StartsAt = startsAt.Time.UTC()
	}
	return round, uint32(earlier + 1), nil //nolint:gosec // a round a day.
}

func (q *PostgresQuery) closed(ctx context.Context) (*seasonsv1.ClosedRound, error) {
	round, number, err := scanRound(q.db.On(ctx).QueryRowContext(ctx, `
		SELECT `+roundColumns+`
		FROM rounds AS round WHERE round.closed ORDER BY round.ends_at DESC, round.own DESC, round.season DESC LIMIT 1
	`))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil //nolint:nilnil // no round closed yet: the race carries none.
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the round closed last: %w", err)
	}

	closed := &seasonsv1.ClosedRound{
		Season:        uint32(round.Season),
		Number:        number,
		EndedAtUnixMs: round.EndsAt.UnixMilli(),
		Finale:        round.Finale,
		Own:           round.Own,
		Times:         round.Times,
	}
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() (err error) {
		closed.Standings, err = q.results(ctx, round)
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
	if round.Own {
		closed.StartedAtUnixMs = round.StartsAt.UnixMilli()
		group.Go(func() (err error) {
			closed.Takes, err = q.takes(ctx, round)
			return err
		})
		group.Go(func() (err error) {
			closed.Players, err = q.players(ctx, round)
			return err
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err //nolint:wrapcheck // each part named what failed.
	}
	return closed, nil
}

func (q *PostgresQuery) results(ctx context.Context, round rounds.Round) ([]*seasonsv1.RoundStanding, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT rank, country, points, share FROM round_results
		WHERE season = $1 AND ends_at = $2 AND own = $3 AND points > 0
		ORDER BY rank, country
	`, int64(round.Season), round.EndsAt, round.Own)
	if err != nil {
		return nil, fmt.Errorf("failed to read the results of the round closed last: %w", err)
	}
	defer func() { _ = rows.Close() }()

	standings := []*seasonsv1.RoundStanding{}
	for rows.Next() {
		var (
			rank, points int64
			standing     = &seasonsv1.RoundStanding{}
		)
		if err := rows.Scan(&rank, &standing.CountryId, &points, &standing.Share); err != nil {
			return nil, fmt.Errorf("failed to read a result of the round closed last: %w", err)
		}
		standing.Rank = uint32(rank)     //nolint:gosec // a few hundred countries at most.
		standing.Points = uint32(points) //nolint:gosec // 250 a round at most.
		standings = append(standings, standing)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the results of the round closed last: %w", err)
	}
	return standings, nil
}

func (q *PostgresQuery) takes(ctx context.Context, round rounds.Round) ([]*seasonsv1.RoundTakes, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT country, tiles FROM round_takes WHERE season = $1 AND ends_at = $2 AND own = $3 ORDER BY tiles DESC, country
	`, int64(round.Season), round.EndsAt, round.Own)
	if err != nil {
		return nil, fmt.Errorf("failed to read the tiles taken in the round closed last: %w", err)
	}
	defer func() { _ = rows.Close() }()

	takes := []*seasonsv1.RoundTakes{}
	for rows.Next() {
		var (
			tiles int64
			taken = &seasonsv1.RoundTakes{}
		)
		if err := rows.Scan(&taken.CountryId, &tiles); err != nil {
			return nil, fmt.Errorf("failed to read the tiles a flag took in the round closed last: %w", err)
		}
		taken.Tiles = uint64(tiles) //nolint:gosec // CHECK (tiles > 0).
		takes = append(takes, taken)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the tiles taken in the round closed last: %w", err)
	}
	return takes, nil
}

type line struct {
	account rounds.AccountID
	country string
	tiles   uint64
}

// players ranks each account by the tiles it took for the flag it took the most for, so it cannot add up several, and
// names the first Shown who are not guests, as the boards do.
func (q *PostgresQuery) players(ctx context.Context, round rounds.Round) ([]*seasonsv1.Standing, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT account_id, country, tiles FROM (
			SELECT DISTINCT ON (account_id) account_id, country, tiles FROM round_players
			WHERE season = $1 AND ends_at = $2 AND own = $3
			ORDER BY account_id, tiles DESC, country
		) AS main ORDER BY tiles DESC, account_id LIMIT $4
	`, int64(round.Season), round.EndsAt, round.Own, read)
	if err != nil {
		return nil, fmt.Errorf("failed to read the players of the round closed last: %w", err)
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
			return nil, fmt.Errorf("failed to read a player of the round closed last: %w", err)
		}
		each.account, each.tiles = rounds.AccountID(account), uint64(tiles) //nolint:gosec // CHECK (tiles > 0).
		lines = append(lines, each)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the players of the round closed last: %w", err)
	}
	return q.named(ctx, lines)
}

func (q *PostgresQuery) named(ctx context.Context, lines []line) ([]*seasonsv1.Standing, error) {
	top := []*seasonsv1.Standing{}
	if len(lines) == 0 {
		return top, nil
	}
	accounts := make([]rounds.AccountID, len(lines))
	for i, line := range lines {
		accounts[i] = line.account
	}
	authors, err := q.authors.Authors(ctx, accounts)
	if err != nil {
		return nil, fmt.Errorf("failed to read who the players of the round closed last are: %w", err)
	}

	for _, line := range lines {
		author, known := authors[line.account]
		if !known || author.GetGuest() {
			continue
		}
		rank := uint32(len(top) + 1) //nolint:gosec // at most Shown.
		if len(top) > 0 && top[len(top)-1].GetTiles() == line.tiles {
			rank = top[len(top)-1].GetRank()
		}
		top = append(top, &seasonsv1.Standing{
			Rank:      rank,
			AccountId: line.account.String(),
			Name:      author.GetName(),
			Color:     author.GetColor(),
			CountryId: line.country,
			Tiles:     line.tiles,
			WornTitle: author.GetWornTitle(),
		})
		if len(top) == Shown {
			break
		}
	}
	return top, nil
}

// round is the round in progress that ends first: the day, or a round of its own, which wins a tie.
func (q *PostgresQuery) round(ctx context.Context, season calendar.Season, now time.Time) (*seasonsv1.Round, error) {
	own, ownNumber, err := scanRound(q.db.On(ctx).QueryRowContext(ctx, `
		SELECT `+roundColumns+`
		FROM rounds AS round WHERE round.own AND NOT round.closed AND round.season = $1 AND round.ends_at > $2
		ORDER BY round.ends_at LIMIT 1
	`, int64(season.Number), now))
	hasOwn := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to read the round of its own in progress: %w", err)
	}

	round, number := own, ownNumber
	if day, ok := rounds.Day(q.seasons, now); ok && (!hasOwn || day.EndsAt.Before(own.EndsAt)) {
		round = day
		if number, err = q.dayNumber(ctx, day); err != nil {
			return nil, err
		}
	} else if !hasOwn {
		return nil, nil //nolint:nilnil // from the last cutoff to the season's end, only a round of its own runs.
	}

	held, err := q.territory.Snapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read what each country holds now: %w", err)
	}
	shown := &seasonsv1.Round{
		Number:       number,
		EndsAtUnixMs: round.EndsAt.UnixMilli(),
		Finale:       round.Finale,
		Standings:    []*seasonsv1.RoundStanding{},
		Own:          round.Own,
		Times:        round.Times,
	}
	for _, result := range round.Results(held) {
		shown.Standings = append(shown.Standings, &seasonsv1.RoundStanding{
			Rank:      result.Rank,
			CountryId: string(result.Country),
			Share:     result.Share,
			Points:    result.Points,
		})
	}
	return shown, nil
}

func (q *PostgresQuery) dayNumber(ctx context.Context, day rounds.Round) (uint32, error) {
	var earlier int64
	if err := q.db.On(ctx).QueryRowContext(ctx, `
		SELECT count(*) FROM rounds WHERE season = $1 AND NOT own AND ends_at < $2
	`, int64(day.Season), day.EndsAt).Scan(&earlier); err != nil {
		return 0, fmt.Errorf("failed to number the day in progress: %w", err)
	}
	return uint32(earlier + 1), nil //nolint:gosec // a round a day.
}

type bound string

const (
	before  bound = "<"
	through bound = "<="
)

func (q *PostgresQuery) scores(ctx context.Context, season calendar.Number, bound bound, at time.Time) ([]*seasonsv1.Score, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT result.country,
		       sum(result.points),
		       count(*) FILTER (WHERE result.rank = 1),
		       coalesce(sum(result.points) FILTER (WHERE round.finale), 0)
		FROM round_results AS result JOIN rounds AS round USING (season, ends_at, own)
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
			Points:    uint32(placed.Points), //nolint:gosec // 250 a round at most.
			RoundsWon: placed.RoundsWon,
		})
	}
	return shown, nil
}

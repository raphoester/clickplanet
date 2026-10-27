package hall_of_fame_query

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

type Authors interface {
	Authors(ctx context.Context, accounts []standings.AccountID) (map[standings.AccountID]*playerv1.Author, error)
}

const (
	Shown = 10
	page  = 200
)

func NewPostgresQuery(db cppg.Conn, authors Authors) *PostgresQuery {
	return &PostgresQuery{db: db, authors: authors}
}

type PostgresQuery struct {
	db      cppg.Conn
	authors Authors
}

func (q *PostgresQuery) HallOfFame(ctx context.Context) (*seasonsv1.GetHallOfFameResponse, error) {
	ended, err := q.ended(ctx)
	if err != nil {
		return nil, err
	}

	group, ctx := errgroup.WithContext(ctx)
	for _, season := range ended {
		group.Go(func() (err error) {
			season.Scores, err = q.scores(ctx, season.GetNumber())
			return err
		})
		group.Go(func() (err error) {
			season.Standings, err = q.top(ctx, season.GetNumber())
			return err
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err //nolint:wrapcheck // each part named what failed.
	}
	return &seasonsv1.GetHallOfFameResponse{Seasons: ended}, nil
}

func (q *PostgresQuery) ended(ctx context.Context) ([]*seasonsv1.EndedSeason, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT season, ended_at FROM ended_seasons ORDER BY season DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the seasons that ended: %w", err)
	}
	defer func() { _ = rows.Close() }()

	ended := []*seasonsv1.EndedSeason{}
	for rows.Next() {
		var (
			season  int64
			endedAt time.Time
		)
		if err := rows.Scan(&season, &endedAt); err != nil {
			return nil, fmt.Errorf("failed to read a season that ended: %w", err)
		}
		ended = append(ended, &seasonsv1.EndedSeason{
			Number:        uint32(season), //nolint:gosec // CHECK (season >= 0).
			EndedAtUnixMs: endedAt.UnixMilli(),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the seasons that ended: %w", err)
	}
	return ended, nil
}

func (q *PostgresQuery) scores(ctx context.Context, season uint32) ([]*seasonsv1.Score, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT result.country,
		       sum(result.points),
		       count(*) FILTER (WHERE result.rank = 1),
		       coalesce(sum(result.points) FILTER (WHERE round.finale), 0)
		FROM round_results AS result JOIN rounds AS round USING (season, ends_at, own)
		WHERE result.season = $1
		GROUP BY result.country
		HAVING sum(result.points) > 0
	`, int64(season))
	if err != nil {
		return nil, fmt.Errorf("failed to read the final table of season %d: %w", season, err)
	}
	defer func() { _ = rows.Close() }()

	var scores []rounds.Score
	for rows.Next() {
		var (
			country                 string
			points, won, finalePart int64
		)
		if err := rows.Scan(&country, &points, &won, &finalePart); err != nil {
			return nil, fmt.Errorf("failed to read a country's points in season %d: %w", season, err)
		}
		scores = append(scores, rounds.Score{
			Country:      rounds.Country(country),
			Points:       uint64(points),     //nolint:gosec // CHECK (points >= 0).
			RoundsWon:    uint32(won),        //nolint:gosec // a round a day.
			FinalePoints: uint64(finalePart), //nolint:gosec // CHECK (points >= 0).
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the final table of season %d: %w", season, err)
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

func (q *PostgresQuery) top(ctx context.Context, season uint32) ([]*seasonsv1.Standing, error) {
	top := make([]*seasonsv1.Standing, 0, Shown)
	after := start
	for {
		lines, err := q.lines(ctx, season, after)
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
			top = append(top, &seasonsv1.Standing{
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

func (q *PostgresQuery) lines(ctx context.Context, season uint32, after line) ([]line, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT account_id, country, tiles
		FROM contributions
		WHERE season = $1 AND main AND (tiles < $2 OR (tiles = $2 AND account_id > $3))
		ORDER BY tiles DESC, account_id
		LIMIT $4
	`, int64(season), int64(after.tiles), uuid.UUID(after.account), page) //nolint:gosec // a line's tiles fit in int64.
	if err != nil {
		return nil, fmt.Errorf("failed to read the players of season %d: %w", season, err)
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
			return nil, fmt.Errorf("failed to scan a player of season %d: %w", season, err)
		}
		each.account = standings.AccountID(account)
		each.tiles = uint64(tiles) //nolint:gosec // CHECK (tiles > 0).
		lines = append(lines, each)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the players of season %d: %w", season, err)
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
		return nil, fmt.Errorf("failed to read who the players are: %w", err)
	}
	return named, nil
}

func rankAfter(top []*seasonsv1.Standing, tiles uint64) uint32 {
	if len(top) > 0 && top[len(top)-1].GetTiles() == tiles {
		return top[len(top)-1].GetRank()
	}
	return uint32(len(top) + 1) //nolint:gosec // at most Shown.
}

package round_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

type Territory interface {
	Snapshot(ctx context.Context) (books.Snapshot, error)
}

type Authors interface {
	Authors(ctx context.Context, accounts []books.AccountID) (map[books.AccountID]*playerv1.Author, error)
}

type Scores interface {
	Around(ctx context.Context, book books.BookID, endedAt time.Time) ([]*scoringv1.Score, []*scoringv1.Score, error)
}

var ErrNoRound = errors.New("no round has this id")

const (
	// Shown is how many players a round names.
	Shown = 10
	// read is how many of its best players are asked for, to name Shown who are not guests.
	read = 50
)

func NewPostgresQuery(db cppg.Conn, territory Territory, authors Authors, scores Scores) *PostgresQuery {
	return &PostgresQuery{db: db, territory: territory, authors: authors, scores: scores}
}

type PostgresQuery struct {
	db        cppg.Conn
	territory Territory
	authors   Authors
	scores    Scores
}

func (q *PostgresQuery) Round(ctx context.Context, id books.RoundID) (*scoringv1.Round, error) {
	round, found, err := q.Live(ctx, id)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("%w: %q", ErrNoRound, id)
	}
	return round, nil
}

// Live is the round a stream shows: none until it is opened.
func (q *PostgresQuery) Live(ctx context.Context, id books.RoundID) (*scoringv1.Round, bool, error) {
	var (
		book     string
		times    int64
		startsAt time.Time
		endedAt  sql.NullTime
	)
	err := q.db.On(ctx).QueryRowContext(ctx, `
		SELECT round.book_id, round.times, round.starts_at, scored.ended_at
		FROM rounds AS round LEFT JOIN scored_rounds AS scored USING (round_id)
		WHERE round.round_id = $1
	`, string(id)).Scan(&book, &times, &startsAt, &endedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("failed to read the round: %w", err)
	}

	shown := &scoringv1.Round{
		RoundId:         string(id),
		BookId:          book,
		Times:           uint32(times), //nolint:gosec // CHECK (times BETWEEN 1 AND 10).
		StartedAtUnixMs: startsAt.UnixMilli(),
	}
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() (err error) {
		shown.Takes, err = q.takes(ctx, id)
		return err
	})
	group.Go(func() (err error) {
		shown.Players, err = q.players(ctx, id)
		return err
	})
	if endedAt.Valid {
		shown.EndedAtUnixMs = endedAt.Time.UnixMilli()
		group.Go(func() (err error) {
			shown.Standings, err = q.results(ctx, id)
			return err
		})
		group.Go(func() (err error) {
			shown.Before, shown.After, err = q.scores.Around(ctx, books.BookID(book), endedAt.Time)
			return err //nolint:wrapcheck // the scores query named it.
		})
	} else {
		group.Go(func() (err error) {
			shown.Standings, err = q.live(ctx, books.Round{ID: id, Times: shown.GetTimes()})
			return err
		})
	}
	if err := group.Wait(); err != nil {
		return nil, false, err //nolint:wrapcheck // each part named what failed.
	}
	return shown, true, nil
}

func (q *PostgresQuery) live(ctx context.Context, round books.Round) ([]*scoringv1.RoundStanding, error) {
	held, err := q.territory.Snapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read what each country holds now: %w", err)
	}
	standings := []*scoringv1.RoundStanding{}
	for _, result := range round.Results(held) {
		standings = append(standings, &scoringv1.RoundStanding{
			Rank:      result.Rank,
			CountryId: string(result.Country),
			Share:     result.Share,
			Points:    result.Points,
		})
	}
	return standings, nil
}

func (q *PostgresQuery) results(ctx context.Context, id books.RoundID) ([]*scoringv1.RoundStanding, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT rank, country, points, share FROM round_results WHERE round_id = $1 AND points > 0 ORDER BY rank, country
	`, string(id))
	if err != nil {
		return nil, fmt.Errorf("failed to read the round's results: %w", err)
	}
	defer func() { _ = rows.Close() }()

	standings := []*scoringv1.RoundStanding{}
	for rows.Next() {
		var (
			rank, points int64
			standing     = &scoringv1.RoundStanding{}
		)
		if err := rows.Scan(&rank, &standing.CountryId, &points, &standing.Share); err != nil {
			return nil, fmt.Errorf("failed to read a result of the round: %w", err)
		}
		standing.Rank = uint32(rank)     //nolint:gosec // a few hundred countries at most.
		standing.Points = uint32(points) //nolint:gosec // 250 a round at most.
		standings = append(standings, standing)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the round's results: %w", err)
	}
	return standings, nil
}

func (q *PostgresQuery) takes(ctx context.Context, id books.RoundID) ([]*scoringv1.RoundTakes, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT country, tiles FROM round_takes WHERE round_id = $1 ORDER BY tiles DESC, country
	`, string(id))
	if err != nil {
		return nil, fmt.Errorf("failed to read the tiles taken in the round: %w", err)
	}
	defer func() { _ = rows.Close() }()

	takes := []*scoringv1.RoundTakes{}
	for rows.Next() {
		var (
			tiles int64
			taken = &scoringv1.RoundTakes{}
		)
		if err := rows.Scan(&taken.CountryId, &tiles); err != nil {
			return nil, fmt.Errorf("failed to read the tiles a flag took in the round: %w", err)
		}
		taken.Tiles = uint64(tiles) //nolint:gosec // CHECK (tiles > 0).
		takes = append(takes, taken)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the tiles taken in the round: %w", err)
	}
	return takes, nil
}

type line struct {
	account books.AccountID
	country string
	tiles   uint64
}

// players ranks each account by the tiles it took for the flag it took the most for, so it cannot add up several, and
// names the first Shown who are not guests, as the boards do.
func (q *PostgresQuery) players(ctx context.Context, id books.RoundID) ([]*scoringv1.Standing, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT account_id, country, tiles FROM (
			SELECT DISTINCT ON (account_id) account_id, country, tiles FROM round_players
			WHERE round_id = $1
			ORDER BY account_id, tiles DESC, country
		) AS main ORDER BY tiles DESC, account_id LIMIT $2
	`, string(id), read)
	if err != nil {
		return nil, fmt.Errorf("failed to read the players of the round: %w", err)
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
			return nil, fmt.Errorf("failed to read a player of the round: %w", err)
		}
		each.account, each.tiles = books.AccountID(account), uint64(tiles) //nolint:gosec // CHECK (tiles > 0).
		lines = append(lines, each)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the players of the round: %w", err)
	}
	return q.named(ctx, lines)
}

func (q *PostgresQuery) named(ctx context.Context, lines []line) ([]*scoringv1.Standing, error) {
	top := []*scoringv1.Standing{}
	if len(lines) == 0 {
		return top, nil
	}
	accounts := make([]books.AccountID, len(lines))
	for i, line := range lines {
		accounts[i] = line.account
	}
	authors, err := q.authors.Authors(ctx, accounts)
	if err != nil {
		return nil, fmt.Errorf("failed to read who the players of the round are: %w", err)
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
		top = append(top, &scoringv1.Standing{
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

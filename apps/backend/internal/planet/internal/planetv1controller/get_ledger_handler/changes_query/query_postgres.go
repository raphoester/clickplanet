package changes_query

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// Past 72 hours the ledger blanks the address of each change, so who made it is no longer known.
const longest = 72 * time.Hour

// The events of a page: on the 1 GB droplet a whole day at once held 100 MB and took 7s.
const PageEvents = 20_000

var (
	ErrInvalidWindow = errors.New("the ledger is read over a since before its until, at most 72 hours apart")
	ErrInvalidPage   = errors.New("not a page this query gave")
)

func NewPostgresQuery(db cppg.Conn, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, clock: clock}
}

type PostgresQuery struct {
	db    cppg.Conn
	clock cptime.Clock
}

// Each change as Event.Show shows it, in its order. A shield a bomb struck is left out: no owner is kept for it.
// From the head on, every change keeps its address, and the primary key finds the page. An event with no change is
// still a row, its tile 0, so the page counts it.
const changes = `
	SELECT e.position, e.taken_at, COALESCE(c.tile, 0), COALESCE(c.owner, ''), COALESCE(c.previous, ''),
		COALESCE(c.clicked, false), COALESCE(c.shields, 0),
		CASE WHEN e.account IS NOT NULL THEN 'account:' || e.account::text ELSE 'scope:' || COALESCE(e.scope, '') END
	FROM (
		SELECT position, kind, tile, country, previous, taken_at, payload, account, scope
		FROM ledger_events
		WHERE position >= GREATEST(COALESCE((SELECT head FROM ledger_head), 0), $3) AND taken_at >= $1 AND taken_at < $2
		ORDER BY position
		LIMIT $4
	) e
	LEFT JOIN LATERAL (
		SELECT e.tile::bigint AS tile, e.country AS owner, e.previous AS previous, true AS clicked, 0 AS shields,
			0::bigint AS place
		WHERE e.kind = 'take'
		UNION ALL
		SELECT e.tile, e.previous, e.previous, true, (e.payload->>'shields')::int, 0
		WHERE e.kind = 'strike'
		UNION ALL
		SELECT (taken->>'tile')::bigint, e.country, COALESCE(taken->>'owner', ''), false, 0, (taken->>'tile')::bigint
		FROM jsonb_array_elements(COALESCE(e.payload->'taken', '[]'::jsonb)) taken
		WHERE e.kind IN ('spread', 'enclose')
		UNION ALL
		SELECT (struck->>'tile')::bigint, COALESCE(struck->>'owner', ''), COALESCE(struck->>'owner', ''), false,
			(struck->>'shields')::int, (struck->>'tile')::bigint
		FROM jsonb_array_elements(COALESCE(e.payload->'struck', '[]'::jsonb)) struck
		WHERE e.kind IN ('spread', 'enclose')
		UNION ALL
		SELECT (cleared->>'tile')::bigint, '', COALESCE(cleared->>'owner', ''), false, 0, place
		FROM jsonb_array_elements(COALESCE(e.payload->'cleared', '[]'::jsonb)) WITH ORDINALITY AS c(cleared, place)
		WHERE e.kind = 'bomb'
		UNION ALL
		SELECT e.tile, e.country, e.country, false, (e.payload->>'shields')::int, 0
		WHERE e.kind = 'shield'
		UNION ALL
		SELECT (raised->>'tile')::bigint, e.country, e.country, false, (raised->>'shields')::int, place
		FROM jsonb_array_elements(e.payload->'raised') WITH ORDINALITY AS r(raised, place)
		WHERE e.kind = 'fortify'
	) c ON true
	ORDER BY e.position, c.place
`

// Unset until is now. An empty page is the first.
func (q *PostgresQuery) Ledger(ctx context.Context, since, until time.Time, page string) (*planetv1.GetLedgerResponse, error) {
	if until.IsZero() {
		until = q.clock.Now()
	}
	if since.IsZero() || !since.Before(until) || until.Sub(since) > longest {
		return nil, ErrInvalidWindow
	}
	from, err := positionOf(page)
	if err != nil {
		return nil, err
	}

	var answer *planetv1.GetLedgerResponse
	err = q.db.InTx(ctx, func(ctx context.Context) error {
		// Postgres compiled a page's plan for 220ms, to run it in 80.
		if _, err := q.db.On(ctx).ExecContext(ctx, `SET LOCAL jit = off`); err != nil {
			return fmt.Errorf("failed to read the ledger: %w", err)
		}
		answer, err = q.page(ctx, since, until, from)
		return err
	})
	if err != nil {
		return nil, err
	}
	return answer, nil
}

func (q *PostgresQuery) page(ctx context.Context, since, until time.Time, from int64) (*planetv1.GetLedgerResponse, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, changes, since, until, from, PageEvents)
	if err != nil {
		return nil, fmt.Errorf("failed to read the ledger: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var (
		answer = &planetv1.GetLedgerResponse{}
		events int
		last   int64 = -1
	)
	for rows.Next() {
		var (
			position int64
			at       time.Time
			update   = &planetv1.TileUpdate{}
			change   = &planetv1.LedgerChange{Update: update}
		)
		if err := rows.Scan(&position, &at, &update.TileId, &update.CountryId, &update.PreviousCountryId, &update.Clicked,
			&update.Shields, &change.By); err != nil {
			return nil, fmt.Errorf("failed to read a change of the ledger: %w", err)
		}
		if position != last {
			events, last = events+1, position
		}
		if update.GetTileId() == 0 {
			continue
		}
		change.At = timestamppb.New(at)
		answer.Changes = append(answer.Changes, change)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the ledger: %w", err)
	}

	if events == PageEvents {
		answer.NextPage = strconv.FormatInt(last+1, 10)
	}
	return answer, nil
}

func positionOf(page string) (int64, error) {
	if page == "" {
		return 0, nil
	}
	position, err := strconv.ParseInt(page, 10, 64)
	if err != nil || position < 0 {
		return 0, ErrInvalidPage
	}
	return position, nil
}

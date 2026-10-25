package changes_query

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// Past 72 hours the ledger blanks the address of each change, so who made it is no longer known.
const longest = 72 * time.Hour

var ErrInvalidWindow = errors.New("the ledger is read over a since before its until, at most 72 hours apart")

func NewPostgresQuery(db cppg.Conn, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, clock: clock}
}

type PostgresQuery struct {
	db    cppg.Conn
	clock cptime.Clock
}

// Each change as Event.Show shows it, in its order. A shield a bomb struck is left out: no owner is kept for it.
// From the head on, every change keeps its address, and the primary key finds the window.
const changes = `
	WITH events AS (
		SELECT position, kind, tile, country, previous, taken_at, payload,
			CASE WHEN account IS NOT NULL THEN 'account:' || account::text ELSE 'scope:' || COALESCE(scope, '') END AS by
		FROM ledger_events
		WHERE position >= COALESCE((SELECT head FROM ledger_head), 0) AND taken_at >= $1 AND taken_at < $2
	)
	SELECT at, tile, owner, previous, clicked, shields, by FROM (
	SELECT taken_at AS at, tile, country AS owner, previous, true AS clicked, 0 AS shields, by, position, 0 AS place
	FROM events WHERE kind = 'take'
	UNION ALL
	SELECT taken_at, tile, previous, previous, true, (payload->>'shields')::int, by, position, 0
	FROM events WHERE kind = 'strike'
	UNION ALL
	SELECT taken_at, (taken->>'tile')::bigint, country, COALESCE(taken->>'owner', ''), false, 0, by, position,
		(taken->>'tile')::bigint
	FROM events, jsonb_array_elements(COALESCE(payload->'taken', '[]'::jsonb)) taken
	WHERE kind IN ('spread', 'enclose')
	UNION ALL
	SELECT taken_at, (struck->>'tile')::bigint, COALESCE(struck->>'owner', ''), COALESCE(struck->>'owner', ''), false,
		(struck->>'shields')::int, by, position, (struck->>'tile')::bigint
	FROM events, jsonb_array_elements(COALESCE(payload->'struck', '[]'::jsonb)) struck
	WHERE kind IN ('spread', 'enclose')
	UNION ALL
	SELECT taken_at, (cleared->>'tile')::bigint, '', COALESCE(cleared->>'owner', ''), false, 0, by, position, place
	FROM events, jsonb_array_elements(COALESCE(payload->'cleared', '[]'::jsonb)) WITH ORDINALITY AS c(cleared, place)
	WHERE kind = 'bomb'
	UNION ALL
	SELECT taken_at, tile, country, country, false, (payload->>'shields')::int, by, position, 0
	FROM events WHERE kind = 'shield'
	UNION ALL
	SELECT taken_at, (raised->>'tile')::bigint, country, country, false, (raised->>'shields')::int, by, position, place
	FROM events, jsonb_array_elements(payload->'raised') WITH ORDINALITY AS r(raised, place)
	WHERE kind = 'fortify'
	) changes ORDER BY position, place
`

// Unset until is now.
func (q *PostgresQuery) Ledger(ctx context.Context, since, until time.Time) (*planetv1.GetLedgerResponse, error) {
	if until.IsZero() {
		until = q.clock.Now()
	}
	if since.IsZero() || !since.Before(until) || until.Sub(since) > longest {
		return nil, ErrInvalidWindow
	}

	rows, err := q.db.On(ctx).QueryContext(ctx, changes, since, until)
	if err != nil {
		return nil, fmt.Errorf("failed to read the ledger: %w", err)
	}
	defer func() { _ = rows.Close() }()

	answer := &planetv1.GetLedgerResponse{}
	for rows.Next() {
		var (
			at     time.Time
			update = &planetv1.TileUpdate{}
			change = &planetv1.LedgerChange{Update: update}
		)
		if err := rows.Scan(&at, &update.TileId, &update.CountryId, &update.PreviousCountryId, &update.Clicked,
			&update.Shields, &change.By); err != nil {
			return nil, fmt.Errorf("failed to read a change of the ledger: %w", err)
		}
		change.At = timestamppb.New(at)
		answer.Changes = append(answer.Changes, change)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the ledger: %w", err)
	}
	return answer, nil
}

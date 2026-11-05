package history_query

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

// From the first event: the map at the window's start is every take before it. Positions follow time, so the read stops
// at the first event at or after the window's end.
const events = `
	SELECT kind, tile, COALESCE(scope, ''), COALESCE(account::text, ''), country, previous, taken_at, payload
	FROM ledger_events
	ORDER BY position
`

func (q *PostgresQuery) History(ctx context.Context, since, until time.Time, step time.Duration) (*planetv1.MapHistory, error) {
	history, err := ledger.NewHistory(since, until, step)
	if err != nil {
		return nil, fmt.Errorf("failed to start the map's history: %w", err)
	}
	resets, err := q.resets(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.db.On(ctx).QueryContext(ctx, events)
	if err != nil {
		return nil, fmt.Errorf("failed to read the ledger: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var entry ledger.Entry
		if err := rows.Scan(&entry.Kind, &entry.Tile, &entry.Scope, &entry.Account, &entry.Country, &entry.Previous,
			&entry.At, &entry.Payload); err != nil {
			return nil, fmt.Errorf("failed to scan an event: %w", err)
		}
		entry.At = entry.At.UTC()
		if history.Done(entry) {
			break
		}
		for len(resets) > 0 && !resets[0].After(entry.At) {
			history.Wipe(resets[0])
			resets = resets[1:]
		}
		if err := history.Record(entry); err != nil {
			return nil, fmt.Errorf("failed to replay an event: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the ledger: %w", err)
	}
	for _, reset := range resets {
		history.Wipe(reset)
	}
	return messageOf(history), nil
}

// The map resets leave no event in the ledger.
func (q *PostgresQuery) resets(ctx context.Context) ([]time.Time, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `SELECT at FROM map_resets ORDER BY at`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the map's resets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var resets []time.Time
	for rows.Next() {
		var at time.Time
		if err := rows.Scan(&at); err != nil {
			return nil, fmt.Errorf("failed to read a reset: %w", err)
		}
		resets = append(resets, at.UTC())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the map's resets: %w", err)
	}
	return resets, nil
}

func messageOf(history *ledger.History) *planetv1.MapHistory {
	answer := &planetv1.MapHistory{}
	for _, frame := range history.Frames() {
		shown := &planetv1.MapFrame{At: timestamppb.New(frame.At), Owners: make([]*planetv1.TileOwner, 0, len(frame.Owners))}
		for _, tile := range slices.Sorted(maps.Keys(frame.Owners)) {
			shown.Owners = append(shown.Owners, &planetv1.TileOwner{Tile: tile, CountryId: frame.Owners[tile]})
		}
		answer.Frames = append(answer.Frames, shown)
	}
	for tile, hits := range history.Hits() {
		answer.Hits = append(answer.Hits, &planetv1.TileHits{Tile: tile, Hits: hits})
	}
	slices.SortFunc(answer.Hits, func(a, b *planetv1.TileHits) int { return cmp.Compare(a.GetTile(), b.GetTile()) })
	return answer
}

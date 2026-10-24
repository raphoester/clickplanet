package snapshots

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	ErrNoSnapshot    = errors.New("a part of a snapshot names no snapshot")
	ErrOtherSnapshot = errors.New("a row of another snapshot")
	ErrUnnamedRow    = errors.New("a row with no post or no platform")
	ErrNotADocument  = errors.New("a row whose document is not a JSON object")
)

type Storage interface {
	// A row or a summary kept already is replaced.
	Keep(ctx context.Context, part Part) error
}

// What one call sends of a snapshot: its summary, and some of its rows. A snapshot comes in parts, each with the
// summary, so one call is never the size of a whole snapshot.
type Part struct {
	takenAt time.Time
	summary json.RawMessage
	rows    []Row
}

type Row struct {
	PostID   string
	Platform string
	// The row as the job wrote it: a JSON object, kept as it came.
	Document json.RawMessage
}

type PartSnapshot struct {
	TakenAt time.Time
	// A clipstats.v1.Snapshot, as protojson writes it.
	Summary json.RawMessage
	Rows    []Row
}

// rowsTakenAt: each row's own time, which has to be the snapshot's.
func NewPart(takenAt time.Time, summary json.RawMessage, rows []Row, rowsTakenAt []time.Time) (Part, error) {
	if takenAt.IsZero() || len(summary) == 0 {
		return Part{}, ErrNoSnapshot
	}
	if len(rowsTakenAt) != len(rows) {
		return Part{}, fmt.Errorf("%d rows for %d times", len(rows), len(rowsTakenAt))
	}
	for i, row := range rows {
		if !rowsTakenAt[i].Equal(takenAt) {
			return Part{}, fmt.Errorf("%w: %s %s was taken at %s, not %s", ErrOtherSnapshot, row.PostID, row.Platform,
				rowsTakenAt[i].UTC().Format(time.RFC3339), takenAt.UTC().Format(time.RFC3339))
		}
		if row.PostID == "" || row.Platform == "" {
			return Part{}, ErrUnnamedRow
		}
		var object map[string]json.RawMessage
		if json.Unmarshal(row.Document, &object) != nil || object == nil {
			return Part{}, fmt.Errorf("%w: %s %s", ErrNotADocument, row.PostID, row.Platform)
		}
	}
	return Part{takenAt: takenAt.UTC(), summary: summary, rows: rows}, nil
}

func (p Part) TakeSnapshot() PartSnapshot {
	return PartSnapshot{TakenAt: p.takenAt, Summary: p.summary, Rows: p.rows}
}

package snapshots_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clipstats/internal/snapshots"
)

var takenAt = time.Date(2026, 10, 25, 5, 41, 7, 0, time.UTC)

func row(post string) snapshots.Row {
	return snapshots.Row{PostID: post, Platform: "tiktok", Document: json.RawMessage(`{"version":1,"analytics":{"views":3}}`)}
}

func TestAPartIsItsSnapshotAndRowsOfIt(t *testing.T) {
	part, err := snapshots.NewPart(takenAt, json.RawMessage(`{"posts":2}`), []snapshots.Row{row("p1"), row("p2")}, []time.Time{takenAt, takenAt})

	require.NoError(t, err)
	kept := part.TakeSnapshot()
	assert.Equal(t, takenAt, kept.TakenAt)
	assert.JSONEq(t, `{"posts":2}`, string(kept.Summary))
	assert.Len(t, kept.Rows, 2)
}

func TestAPartWithNoRowIsASnapshotThatKeptNone(t *testing.T) {
	_, err := snapshots.NewPart(takenAt, json.RawMessage(`{}`), nil, nil)

	require.NoError(t, err)
}

func TestAPartThatIsNotWholeIsRefused(t *testing.T) {
	unnamed := row("")
	text := row("p1")
	text.Document = json.RawMessage(`"views: 3"`)
	broken := row("p1")
	broken.Document = json.RawMessage(`{"views":`)

	for _, refused := range []struct {
		want    error
		takenAt time.Time
		row     *snapshots.Row
		rowAt   time.Time
	}{
		{want: snapshots.ErrNoSnapshot},
		{want: snapshots.ErrOtherSnapshot, takenAt: takenAt, row: new(row("p1")), rowAt: takenAt.Add(time.Second)},
		{want: snapshots.ErrUnnamedRow, takenAt: takenAt, row: &unnamed, rowAt: takenAt},
		{want: snapshots.ErrNotADocument, takenAt: takenAt, row: &text, rowAt: takenAt},
		{want: snapshots.ErrNotADocument, takenAt: takenAt, row: &broken, rowAt: takenAt},
	} {
		var rows []snapshots.Row
		var times []time.Time
		if refused.row != nil {
			rows, times = []snapshots.Row{*refused.row}, []time.Time{refused.rowAt}
		}

		_, err := snapshots.NewPart(refused.takenAt, json.RawMessage(`{}`), rows, times)

		assert.ErrorIs(t, err, refused.want)
	}
}

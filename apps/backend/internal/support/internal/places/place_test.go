package places_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/places"
)

var (
	ada       = places.AccountID{15: 1}
	report    = places.ReportID{15: 1}
	createdAt = time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
)

func TestAReportKeepsWhoSentItWhereAndWhy(t *testing.T) {
	kept, err := places.NewReport(report, ada, 42, places.Complaint{Reason: places.ReasonHate, Text: "  a swastika \r\n"}, createdAt)
	require.NoError(t, err)

	assert.Equal(t, places.Snapshot{
		ID: report, Reporter: ada, Tile: 42, Complaint: places.Complaint{Reason: places.ReasonHate, Text: "a swastika"},
		At: createdAt,
	}, kept.TakeSnapshot())
}

func TestTheTextIsOptional(t *testing.T) {
	kept, err := places.NewReport(report, ada, 42, places.Complaint{Reason: places.ReasonOther, Text: " \n "}, createdAt)
	require.NoError(t, err)

	assert.Empty(t, kept.TakeSnapshot().Complaint.Text)
}

func TestAnAnonymizedReportKeepsAllButItsReporter(t *testing.T) {
	kept, err := places.NewReport(report, ada, 42, places.Complaint{Reason: places.ReasonSexual}, createdAt)
	require.NoError(t, err)

	anonymized := places.AllAnonymized([]places.Report{kept})

	require.Len(t, anonymized, 1)
	assert.Equal(t, places.Snapshot{
		ID: report, Tile: 42, Complaint: places.Complaint{Reason: places.ReasonSexual}, At: createdAt,
	}, anonymized[0].TakeSnapshot())
}

func TestAReportSentHasAReporterAndOneReadBackMayNot(t *testing.T) {
	_, err := places.NewReport(report, places.AccountID{}, 42, places.Complaint{Reason: places.ReasonHate}, createdAt)
	require.ErrorIs(t, err, places.ErrInvalidReport)

	_, err = places.Snapshot{ID: report, Tile: 42, Complaint: places.Complaint{Reason: places.ReasonHate}, At: createdAt}.Restore()
	require.NoError(t, err)
}

func TestASnapshotThatBreaksARuleRestoresNothing(t *testing.T) {
	valid := places.Snapshot{ID: report, Reporter: ada, Tile: 42, Complaint: places.Complaint{Reason: places.ReasonHate}, At: createdAt}

	for name, broken := range map[string]struct {
		change func(places.Snapshot) places.Snapshot
		err    error
	}{
		"no id":   {func(s places.Snapshot) places.Snapshot { s.ID = places.ReportID{}; return s }, places.ErrInvalidReport},
		"no tile": {func(s places.Snapshot) places.Snapshot { s.Tile = 0; return s }, places.ErrInvalidReport},
		"no time": {func(s places.Snapshot) places.Snapshot { s.At = time.Time{}; return s }, places.ErrInvalidReport},
		"no reason": {
			func(s places.Snapshot) places.Snapshot { s.Complaint.Reason = ""; return s }, places.ErrUnknownReason,
		},
		"a reason nobody named": {
			func(s places.Snapshot) places.Snapshot { s.Complaint.Reason = "ugly"; return s }, places.ErrUnknownReason,
		},
		"a text too long": {
			func(s places.Snapshot) places.Snapshot {
				s.Complaint.Text = strings.Repeat("a", notes.MaxTextLength+1)
				return s
			},
			notes.ErrInvalidText,
		},
	} {
		_, err := broken.change(valid).Restore()
		require.ErrorIs(t, err, broken.err, name)
	}
}

func TestAReportIDIsAUUIDAndNothingElse(t *testing.T) {
	id, err := places.ReportIDOf("00000000-0000-0000-0000-000000000001")
	require.NoError(t, err)
	assert.Equal(t, report, id)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", id.String())

	for _, value := range []string{"", "not-a-report", "00000000-0000-0000-0000-000000000000"} {
		_, err := places.ReportIDOf(value)
		require.ErrorIs(t, err, places.ErrInvalidReportID, value)
	}
}

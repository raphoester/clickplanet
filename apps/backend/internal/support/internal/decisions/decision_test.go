package decisions_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
)

var (
	report    = reports.ReportID{15: 1}
	decidedAt = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
)

func TestADecisionKeepsWhatTheOperatorSaid(t *testing.T) {
	decision, err := decisions.NewDecision(report, decisions.OutcomeActed, "  muted for a day\r\n", decidedAt)
	require.NoError(t, err)

	assert.Equal(t, decisions.Snapshot{
		Report: report, Outcome: decisions.OutcomeActed, Comment: "muted for a day", At: decidedAt,
	}, decision.TakeSnapshot())
}

func TestTheCommentIsOptional(t *testing.T) {
	decision, err := decisions.NewDecision(report, decisions.OutcomeDismissed, " \n ", decidedAt)
	require.NoError(t, err)

	assert.Empty(t, decision.TakeSnapshot().Comment)
}

func TestASnapshotRestoresTheDecisionItWasTakenFrom(t *testing.T) {
	kept := decisions.Snapshot{Report: report, Outcome: decisions.OutcomeDismissed, Comment: "a duel", At: decidedAt}

	decision, err := kept.Restore()
	require.NoError(t, err)

	assert.Equal(t, kept, decision.TakeSnapshot())
}

func TestADecisionThatBreaksARuleRestoresNothing(t *testing.T) {
	for name, broken := range map[string]struct {
		snapshot decisions.Snapshot
		err      error
	}{
		"no report": {
			decisions.Snapshot{Outcome: decisions.OutcomeActed, At: decidedAt}, decisions.ErrInvalidDecision,
		},
		"no time": {
			decisions.Snapshot{Report: report, Outcome: decisions.OutcomeActed}, decisions.ErrInvalidDecision,
		},
		"no outcome": {
			decisions.Snapshot{Report: report, At: decidedAt}, decisions.ErrUnknownOutcome,
		},
		"an outcome nobody named": {
			decisions.Snapshot{Report: report, Outcome: "banned", At: decidedAt}, decisions.ErrUnknownOutcome,
		},
		"a comment too long": {
			decisions.Snapshot{
				Report: report, Outcome: decisions.OutcomeActed, Comment: strings.Repeat("a", notes.MaxTextLength+1), At: decidedAt,
			},
			notes.ErrInvalidText,
		},
	} {
		_, err := broken.snapshot.Restore()
		require.ErrorIs(t, err, broken.err, name)
	}
}

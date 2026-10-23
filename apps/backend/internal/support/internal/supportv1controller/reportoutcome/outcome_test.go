package reportoutcome_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/reportoutcome"
)

func TestEveryOutcomeTheProtoNamesIsAnOutcomeOfADecision(t *testing.T) {
	for value, name := range supportv1.ReportOutcome_name {
		outcome := reportoutcome.Of(supportv1.ReportOutcome(value))
		_, err := decisions.NewDecision(reports.ReportID{15: 1}, outcome, "", time.Now())

		if supportv1.ReportOutcome(value) == supportv1.ReportOutcome_REPORT_OUTCOME_UNSPECIFIED {
			require.ErrorIs(t, err, decisions.ErrUnknownOutcome, name)
			continue
		}
		require.NoError(t, err, name)
		assert.NotEmpty(t, outcome, name)
	}
}

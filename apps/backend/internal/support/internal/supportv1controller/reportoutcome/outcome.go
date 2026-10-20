package reportoutcome

import (
	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions"
)

var outcomes = map[supportv1.ReportOutcome]decisions.Outcome{
	supportv1.ReportOutcome_REPORT_OUTCOME_ACTED:     decisions.OutcomeActed,
	supportv1.ReportOutcome_REPORT_OUTCOME_DISMISSED: decisions.OutcomeDismissed,
}

// The wire's outcome as the decision's. One the wire does not name is none, which a decision refuses.
func Of(outcome supportv1.ReportOutcome) decisions.Outcome {
	return outcomes[outcome]
}

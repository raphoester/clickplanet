package decisions

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
)

var (
	ErrInvalidDecision = errors.New("the decision is invalid")
	ErrUnknownOutcome  = errors.New("not an outcome of a report")
	ErrUnknownReport   = errors.New("no report kept under this id")
)

type Outcome string

const (
	OutcomeActed     Outcome = "acted"
	OutcomeDismissed Outcome = "dismissed"
)

func (o Outcome) known() bool {
	switch o {
	case OutcomeActed, OutcomeDismissed:
		return true
	default:
		return false
	}
}

// What the operator did about a report, and why. It goes with its report.
type Decision struct {
	report  reports.ReportID
	outcome Outcome
	comment string
	at      time.Time
}

func NewDecision(report reports.ReportID, outcome Outcome, comment string, at time.Time) (Decision, error) {
	return Snapshot{Report: report, Outcome: outcome, Comment: comment, At: at}.Restore()
}

func (d Decision) TakeSnapshot() Snapshot {
	return Snapshot{Report: d.report, Outcome: d.outcome, Comment: d.comment, At: d.at}
}

// The state of a Decision, for a store. It holds no rule: Restore does.
type Snapshot struct {
	Report  reports.ReportID
	Outcome Outcome
	// Optional.
	Comment string
	At      time.Time
}

func (s Snapshot) Restore() (Decision, error) {
	switch {
	case s.Report == reports.ReportID{}:
		return Decision{}, fmt.Errorf("%w: no report", ErrInvalidDecision)
	case s.At.IsZero():
		return Decision{}, fmt.Errorf("%w: no time", ErrInvalidDecision)
	case !s.Outcome.known():
		return Decision{}, fmt.Errorf("%w: %q", ErrUnknownOutcome, string(s.Outcome))
	}

	comment, err := commentOf(s.Comment)
	if err != nil {
		return Decision{}, err
	}

	return Decision{report: s.Report, outcome: s.Outcome, comment: comment, at: s.At}, nil
}

func commentOf(text string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return "", nil
	}

	checked, err := notes.TextOf(text)
	if err != nil {
		return "", fmt.Errorf("failed to check the comment of the decision: %w", err)
	}
	return checked, nil
}

// A report decided again keeps the last decision.
type Storage interface {
	// ErrUnknownReport for a report no longer kept.
	Save(ctx context.Context, decision Decision) error
}

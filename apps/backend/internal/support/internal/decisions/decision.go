package decisions

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
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

// What the operator did about a report, of a player or of a place (R, its id), and why. It goes with its report.
type Decision[R comparable] struct {
	report  R
	outcome Outcome
	comment string
	at      time.Time
}

func NewDecision[R comparable](report R, outcome Outcome, comment string, at time.Time) (Decision[R], error) {
	return Snapshot[R]{Report: report, Outcome: outcome, Comment: comment, At: at}.Restore()
}

func (d Decision[R]) TakeSnapshot() Snapshot[R] {
	return Snapshot[R]{Report: d.report, Outcome: d.outcome, Comment: d.comment, At: d.at}
}

// The state of a Decision, for a store. It holds no rule: Restore does.
type Snapshot[R comparable] struct {
	Report  R
	Outcome Outcome
	// Optional.
	Comment string
	At      time.Time
}

func (s Snapshot[R]) Restore() (Decision[R], error) {
	var none R
	switch {
	case s.Report == none:
		return Decision[R]{}, fmt.Errorf("%w: no report", ErrInvalidDecision)
	case s.At.IsZero():
		return Decision[R]{}, fmt.Errorf("%w: no time", ErrInvalidDecision)
	case !s.Outcome.known():
		return Decision[R]{}, fmt.Errorf("%w: %q", ErrUnknownOutcome, string(s.Outcome))
	}

	comment, err := commentOf(s.Comment)
	if err != nil {
		return Decision[R]{}, err
	}

	return Decision[R]{report: s.Report, outcome: s.Outcome, comment: comment, at: s.At}, nil
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
type Storage[R comparable] interface {
	// ErrUnknownReport for a report no longer kept.
	Save(ctx context.Context, decision Decision[R]) error
}

// Package appeals is a player's ask that a person look again at the judge's rejection of its submission: asked once,
// the submission waiting again for the operator, then answered by the operator's verdict, which is final.
package appeals

import (
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

var (
	ErrNotTheJudges  = errors.New("only a rejection the judge made is looked at again by a person")
	ErrAskedAlready  = errors.New("a person was asked to look at this submission again already")
	ErrInvalidAppeal = errors.New("an appeal kept in no state it can be in")
)

type State string

const (
	// The submission waits again, for the operator.
	Asked State = "asked"
	// The operator approved or rejected it since.
	Answered State = "answered"
)

// Asking is what an appeal reads: who asks, the submission, the judge's judgement of it, every submission of the
// account (one waits at a time), and the submission's appeals before (one at most).
type Asking struct {
	Account    catalog.AccountID
	Submission submissions.Submission
	// The zero judgement for one the judge never judged.
	Judgement judgements.Judgement
	Founded   []submissions.Submission
	Earlier   []Appeal
}

type Appeal struct {
	submission submissions.SubmissionID
	account    catalog.AccountID
	// The judge's rejection it asks about.
	rejection submissions.Rejection
	state     State
	askedAt   time.Time
	decidedAt time.Time
}

// Ask is a founder's appeal of the judge's rejection, once per submission: the submission waits again for a person,
// with no payment, and the operator is told.
func Ask(asking Asking, at time.Time) (Appeal, submissions.Submission, []proto.Message, error) {
	kept := asking.Submission.TakeSnapshot()
	if len(asking.Earlier) > 0 {
		return Appeal{}, asking.Submission, nil, fmt.Errorf("%w: %s", ErrAskedAlready, kept.ID)
	}
	if !asking.Judgement.Rejected() {
		return Appeal{}, asking.Submission, nil, fmt.Errorf("%w: %s", ErrNotTheJudges, kept.ID)
	}
	reopened, err := asking.Submission.Reopened(asking.Account, asking.Founded, at)
	if err != nil {
		return Appeal{}, asking.Submission, nil, err //nolint:wrapcheck // the submission names it.
	}
	appeal := Appeal{
		submission: kept.ID, account: asking.Account, rejection: kept.Rejection, state: Asked, askedAt: at, decidedAt: at,
	}
	told := asking.Judgement.ReviewAsked(reopened, flagsv1.ReviewCause_REVIEW_CAUSE_APPEAL, at)
	return appeal, reopened, []proto.Message{told}, nil
}

// Answered is the appeal once the operator decided the submission at a time: a verdict from before the ask, the
// judge's rejection itself heard late, answers nothing. False when nothing changed.
func (a Appeal) Answered(at time.Time) (Appeal, bool) {
	if a.state != Asked || at.Before(a.askedAt) {
		return a, false
	}
	a.state, a.decidedAt = Answered, at
	return a, true
}

func (a Appeal) TakeSnapshot() Snapshot {
	return Snapshot{
		Submission: a.submission, Account: a.account, Rejection: a.rejection, State: a.state, AskedAt: a.askedAt,
		DecidedAt: a.decidedAt,
	}
}

type Snapshot struct {
	Submission submissions.SubmissionID
	Account    catalog.AccountID
	Rejection  submissions.Rejection
	State      State
	AskedAt    time.Time
	// When it took its state.
	DecidedAt time.Time
}

func (s Snapshot) Restore() (Appeal, error) {
	switch s.State {
	case Asked, Answered:
	default:
		return Appeal{}, fmt.Errorf("%w: the state %q", ErrInvalidAppeal, s.State)
	}
	if s.Submission == (submissions.SubmissionID{}) || s.Account == (catalog.AccountID{}) {
		return Appeal{}, fmt.Errorf("%w: no submission or no account", ErrInvalidAppeal)
	}
	if _, err := submissions.ParseRejection(s.Rejection.Reason, s.Rejection.Note); err != nil {
		return Appeal{}, fmt.Errorf("%w: %w", ErrInvalidAppeal, err)
	}
	return Appeal{
		submission: s.Submission, account: s.Account, rejection: s.Rejection, state: s.State, askedAt: s.AskedAt,
		decidedAt: s.DecidedAt,
	}, nil
}

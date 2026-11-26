// Package judgements is the review of a submission by a Claude model, as it is sent: an approval or a rejection it
// applies, or unsure, which leaves the submission to the operator.
package judgements

import (
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

var (
	ErrNotWaiting       = errors.New("only a submission that waits is judged")
	ErrInvalidJudgement = errors.New("a judgement kept in no state it can be in")
)

type Verdict string

const (
	Approve Verdict = "approve"
	Reject  Verdict = "reject"
	// Not sure enough to decide: the operator does.
	Unsure Verdict = "unsure"
)

// Case is what the judge sees of a submission.
type Case struct {
	Name catalog.Name
	Pin  catalog.Pin
	// The picture as the server drew it again, a PNG.
	Picture []byte
	// The names of the flags opened since the game shipped, open or closed at their end, never removed: a place has one
	// flag.
	Taken []catalog.Name
}

// Ruling is the judge's answer to a case.
type Ruling struct {
	Verdict Verdict
	// A rejection's reason and the note its player reads.
	Rejection submissions.Rejection
	// For the operator: what decided it.
	Why string
}

// ParseRuling reads the judge's answer: one the game cannot act on is unsure.
func ParseRuling(verdict, reason, note, why string) Ruling {
	switch Verdict(verdict) {
	case Approve:
		return Ruling{Verdict: Approve, Why: why}
	case Reject:
		rejection, err := submissions.ParseRejection(submissions.Reason(reason), note)
		if err != nil {
			return Ruling{Verdict: Unsure, Why: "a rejection the game cannot read: " + why}
		}
		return Ruling{Verdict: Reject, Rejection: rejection, Why: why}
	case Unsure:
	}
	return Ruling{Verdict: Unsure, Why: why}
}

// Judgement is a ruling kept on a submission, for the operator's audit.
type Judgement struct {
	submission submissions.SubmissionID
	verdict    Verdict
	why        string
	judgedAt   time.Time
}

func Judged(submission submissions.SubmissionID, ruling Ruling, at time.Time) Judgement {
	return Judgement{submission: submission, verdict: ruling.Verdict, why: ruling.Why, judgedAt: at}
}

// Applied is the judgement once its verdict was applied, and whether to keep it: a verdict the game refused (a name
// another flag took since) leaves the submission to the operator, and one decided or replaced meanwhile keeps nothing.
func (j Judgement) Applied(err error) (Judgement, bool, error) {
	switch {
	case err == nil:
		return j, true, nil
	case errors.Is(err, submissions.ErrDecided), errors.Is(err, submissions.ErrUnknownSubmission):
		return j, false, nil
	case errors.Is(err, catalog.ErrNameTaken):
		j.verdict, j.why = Unsure, "another flag has its name: "+j.why
		return j, true, nil
	}
	return j, false, err
}

func (j Judgement) TakeSnapshot() Snapshot {
	return Snapshot{Submission: j.submission, Verdict: j.verdict, Why: j.why, JudgedAt: j.judgedAt}
}

type Snapshot struct {
	Submission submissions.SubmissionID
	Verdict    Verdict
	Why        string
	JudgedAt   time.Time
}

func (s Snapshot) Restore() (Judgement, error) {
	switch s.Verdict {
	case Approve, Reject, Unsure:
	default:
		return Judgement{}, fmt.Errorf("%w: the verdict %q", ErrInvalidJudgement, s.Verdict)
	}
	if s.Submission == (submissions.SubmissionID{}) {
		return Judgement{}, fmt.Errorf("%w: no submission", ErrInvalidJudgement)
	}
	return Judgement{submission: s.Submission, verdict: s.Verdict, why: s.Why, judgedAt: s.JudgedAt}, nil
}

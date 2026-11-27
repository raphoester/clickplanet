// Package judgements is the review of a submission by a Claude model, as it is sent: an approval or a rejection it
// applies, or unsure, which leaves the submission to the operator.
package judgements

import (
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

var (
	ErrNotWaiting       = errors.New("only a submission that waits is judged")
	ErrInvalidJudgement = errors.New("a judgement kept in no state it can be in")
	ErrUnjudged         = errors.New("the judge never judged this submission")
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

// Rejected is true when the judge rejected the submission: its player may ask a person to look again.
func (j Judgement) Rejected() bool {
	return j.verdict == Reject
}

// Facts is what the judgement tells of its submission: one left unsure that waits paid for waits on the operator. The
// judgement and the payment each ask it as they land, so whichever comes last tells it.
func (j Judgement) Facts(submission submissions.Submission, at time.Time) []proto.Message {
	if j.verdict != Unsure || !submission.WaitsPaidFor() {
		return nil
	}
	return []proto.Message{j.ReviewAsked(submission, flagsv1.ReviewCause_REVIEW_CAUSE_UNSURE, at)}
}

// ReviewAsked tells the operator a submission waits on a person, with what decided the judge.
func (j Judgement) ReviewAsked(
	submission submissions.Submission, cause flagsv1.ReviewCause, at time.Time,
) *flagsv1.ReviewAsked {
	kept := submission.TakeSnapshot()
	return &flagsv1.ReviewAsked{
		SubmissionId: kept.ID.String(), Name: kept.Name.String(), Cause: cause, Why: j.why, AskedAt: timestamppb.New(at),
	}
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

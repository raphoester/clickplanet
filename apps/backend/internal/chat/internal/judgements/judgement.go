package judgements

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

// What became of a report.
type Verdict uint8

const (
	// Something was removed, and maybe muted: the report is closed as acted.
	Acted Verdict = iota + 1
	// Nothing to act on: the report is closed as dismissed.
	Dismissed
	// The operator's: the report stays open.
	LeftOpen
)

// As postgres keeps it: a value of the type judgement_verdict.
func (v Verdict) String() string {
	switch v {
	case Acted:
		return "acted"
	case Dismissed:
		return "dismissed"
	case LeftOpen:
		return "left_open"
	default:
		return ""
	}
}

func VerdictOf(kept string) (Verdict, error) {
	for _, verdict := range []Verdict{Acted, Dismissed, LeftOpen} {
		if verdict.String() == kept {
			return verdict, nil
		}
	}
	return 0, fmt.Errorf("%w: %q", ErrUnknownVerdict, kept)
}

var (
	ErrNotJudged      = errors.New("no judgement is kept")
	ErrUnknownVerdict = errors.New("not a verdict")
	// The judge is off: no ANTHROPIC_API_KEY.
	ErrNoJudge = errors.New("no judge")
	// The judge would not judge this report, and would not the next time: the model refused it, or the API the request.
	ErrUnjudgeable = errors.New("the judge cannot judge it")
)

// One report's judgement, kept so that a report told again is judged once, and a message reported many times once.
type Judgement struct {
	snapshot Snapshot
}

type Snapshot struct {
	Report ReportID
	// The message reported, NoMessage for a report of a player.
	Message messages.MessageID
	Verdict Verdict
	// The message was judged, by the model or by what became of it (removed, pruned): every later report of it takes
	// this judgement. One left open before anything was judged (the cap spent, no key, another's report followed) does
	// not stand for the next.
	Judged bool
	// The operator should look: a severe offence, one after a 24-hour mute, child safety, self-harm, cheating, a doubt,
	// the cap spent.
	Flagged bool
	// One line, as the report's decision says it.
	Reason string
	At     time.Time
}

func (s Snapshot) Restore() (Judgement, error) {
	switch {
	case s.Report == ReportID{}:
		return Judgement{}, fmt.Errorf("%w: no report", ErrInvalidReport)
	case s.Verdict.String() == "":
		return Judgement{}, fmt.Errorf("%w: %d", ErrUnknownVerdict, s.Verdict)
	case s.At.IsZero():
		return Judgement{}, fmt.Errorf("%w: no time", ErrInvalidReport)
	}
	return Judgement{snapshot: s}, nil
}

func (j Judgement) TakeSnapshot() Snapshot { return j.snapshot }

// What support keeps as the decision's comment: the reason, and who decided it.
func (j Judgement) Comment() string {
	return Client + ": " + j.snapshot.Reason
}

// Another report of a message judged already: the same verdict, decided again with its reason, and never judged again.
func (j Judgement) Followed(report Report, at time.Time) Judgement {
	return Judgement{snapshot: Snapshot{
		Report: report.ID, Message: report.Message, Verdict: j.snapshot.Verdict, Flagged: j.snapshot.Flagged,
		Reason: "as report " + j.snapshot.Report.String() + ": " + j.snapshot.Reason, At: at,
	}}
}

type Storage interface {
	// ErrNotJudged.
	OfReport(ctx context.Context, report ReportID) (Judgement, error)
	// The first judgement that judged the message. ErrNotJudged.
	OfMessage(ctx context.Context, message messages.MessageID) (Judgement, error)
	// A report judged twice keeps the first.
	Save(ctx context.Context, judgement Judgement) error
	DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

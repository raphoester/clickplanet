package reports

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type AccountID = notes.AccountID

type ReportID uuid.UUID

var ErrInvalidReportID = errors.New("not a report id")

func ReportIDOf(value string) (ReportID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return ReportID{}, fmt.Errorf("%w: %q", ErrInvalidReportID, value)
	}
	return ReportID(id), nil
}

var (
	ErrInvalidReport   = errors.New("the report is invalid")
	ErrUnknownReason   = errors.New("not a reason to report a player")
	ErrOwnAccount      = errors.New("a player cannot report itself")
	ErrReportedAlready = errors.New("this account already reported this player today")
)

type Reason string

const (
	ReasonCheating  Reason = "cheating"
	ReasonChatAbuse Reason = "chat_abuse"
	ReasonBadName   Reason = "bad_name"
	ReasonOther     Reason = "other"
)

func (r Reason) known() bool {
	switch r {
	case ReasonCheating, ReasonChatAbuse, ReasonBadName, ReasonOther:
		return true
	default:
		return false
	}
}

const maxMessageIDLength = 64

type MessageID string

const NoMessage MessageID = ""

// What a reporter says of a player. Plain data, a part of a report's snapshot: Restore checks it.
type Complaint struct {
	Reason Reason
	// Optional.
	Text string
	// Optional: the chat message the report is about.
	Message MessageID
}

func (c Complaint) checked() (Complaint, error) {
	if !c.Reason.known() {
		return Complaint{}, fmt.Errorf("%w: %q", ErrUnknownReason, string(c.Reason))
	}

	text, err := comment(c.Text)
	if err != nil {
		return Complaint{}, err
	}

	return Complaint{Reason: c.Reason, Text: text, Message: cleaned(c.Message)}, nil
}

func comment(text string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return "", nil
	}

	checked, err := notes.TextOf(text)
	if err != nil {
		return "", fmt.Errorf("failed to check the text of the report: %w", err)
	}
	return checked, nil
}

func cleaned(message MessageID) MessageID {
	printable := []rune(strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, strings.ToValidUTF8(string(message), ""))))

	if len(printable) > maxMessageIDLength {
		return NoMessage
	}
	return MessageID(printable)
}

type Report struct {
	id        ReportID
	reporter  AccountID
	reported  AccountID
	complaint Complaint
	at        time.Time
	forgotten bool
}

// One account reports one player once a UTC day: earlier is what the reporter already filed, and a report of
// the same player the same day among them is ErrReportedAlready.
func NewReport(
	id ReportID, reporter AccountID, reported AccountID, complaint Complaint, at time.Time, earlier []Report,
) (Report, error) {
	report, err := Snapshot{ID: id, Reporter: reporter, Reported: reported, Complaint: complaint, At: at}.Restore()
	if err != nil {
		return Report{}, err
	}

	if slices.ContainsFunc(earlier, report.repeats) {
		return Report{}, ErrReportedAlready
	}
	return report, nil
}

func (r Report) repeats(earlier Report) bool {
	return earlier.reporter == r.reporter && earlier.reported == r.reported && dayOf(earlier.at).Equal(dayOf(r.at))
}

func dayOf(at time.Time) time.Time {
	year, month, day := at.UTC().Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// A report the deletion of an account it names, or its age, took away: a storage keeps it no more.
func (r Report) Forgotten() Report {
	r.forgotten = true
	return r
}

func AllForgotten(filed []Report) []Report {
	forgotten := make([]Report, 0, len(filed))
	for _, report := range filed {
		forgotten = append(forgotten, report.Forgotten())
	}
	return forgotten
}

func (r Report) TakeSnapshot() Snapshot {
	return Snapshot{
		ID: r.id, Reporter: r.reporter, Reported: r.reported, Complaint: r.complaint, At: r.at, Forgotten: r.forgotten,
	}
}

// The state of a Report, for a store. It holds no rule: Restore does.
type Snapshot struct {
	ID        ReportID
	Reporter  AccountID
	Reported  AccountID
	Complaint Complaint
	At        time.Time
	Forgotten bool
}

func (s Snapshot) Restore() (Report, error) {
	switch {
	case s.ID == ReportID{}:
		return Report{}, fmt.Errorf("%w: no id", ErrInvalidReport)
	case s.Reporter == AccountID{}:
		return Report{}, fmt.Errorf("%w: no reporter", ErrInvalidReport)
	case s.Reported == AccountID{}:
		return Report{}, fmt.Errorf("%w: no player reported", ErrInvalidReport)
	case s.At.IsZero():
		return Report{}, fmt.Errorf("%w: no time", ErrInvalidReport)
	case s.Reporter == s.Reported:
		return Report{}, ErrOwnAccount
	}

	complaint, err := s.Complaint.checked()
	if err != nil {
		return Report{}, err
	}

	return Report{
		id: s.ID, reporter: s.Reporter, reported: s.Reported, complaint: complaint, at: s.At, forgotten: s.Forgotten,
	}, nil
}

// A storage finds reports and saves them. It holds no rule: what changes a report is a method of Report.
type Storage interface {
	// A report saved again replaces itself, and a forgotten report is kept no more. All of them, or none.
	Save(ctx context.Context, reports ...Report) error
	// What a reporter filed about a player, oldest first.
	FiledBy(ctx context.Context, reporter AccountID, about AccountID) ([]Report, error)
	// The reports an account filed and the reports about it, oldest first.
	Naming(ctx context.Context, account AccountID) ([]Report, error)
	// Oldest first, limit at most.
	FiledBefore(ctx context.Context, cutoff time.Time, limit int) ([]Report, error)
}

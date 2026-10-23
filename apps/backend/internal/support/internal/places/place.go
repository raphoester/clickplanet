package places

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type AccountID = notes.AccountID

type ReportID uuid.UUID

func (id ReportID) String() string {
	return uuid.UUID(id).String()
}

// A tile of the drawing reported. The support module does not know the map: the operator's tools check it.
type TileID uint32

var (
	ErrInvalidReportID = errors.New("not a report id")
	ErrInvalidReport   = errors.New("the report is invalid")
	ErrUnknownReason   = errors.New("not a reason to report a place")
)

func ReportIDOf(value string) (ReportID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return ReportID{}, fmt.Errorf("%w: %q", ErrInvalidReportID, value)
	}
	return ReportID(id), nil
}

type Reason string

const (
	ReasonHate   Reason = "hate"
	ReasonSexual Reason = "sexual"
	ReasonOther  Reason = "other"
)

func (r Reason) known() bool {
	switch r {
	case ReasonHate, ReasonSexual, ReasonOther:
		return true
	default:
		return false
	}
}

// What a reporter says of a place. Plain data, a part of a report's snapshot: Restore checks it.
type Complaint struct {
	Reason Reason
	// Optional.
	Text string
}

func (c Complaint) checked() (Complaint, error) {
	if !c.Reason.known() {
		return Complaint{}, fmt.Errorf("%w: %q", ErrUnknownReason, string(c.Reason))
	}
	if strings.TrimSpace(c.Text) == "" {
		return Complaint{Reason: c.Reason}, nil
	}

	text, err := notes.TextOf(c.Text)
	if err != nil {
		return Complaint{}, fmt.Errorf("failed to check the text of the report: %w", err)
	}
	return Complaint{Reason: c.Reason, Text: text}, nil
}

// A report of a drawing on the map, by a tile of it. Who sent it goes with its account, and past the retention: the
// report stays, a notice the operator acted on or not.
type Report struct {
	id        ReportID
	reporter  AccountID
	tile      TileID
	complaint Complaint
	at        time.Time
}

func NewReport(id ReportID, reporter AccountID, tile TileID, complaint Complaint, at time.Time) (Report, error) {
	if reporter == (AccountID{}) {
		return Report{}, fmt.Errorf("%w: no reporter", ErrInvalidReport)
	}
	return Snapshot{ID: id, Reporter: reporter, Tile: tile, Complaint: complaint, At: at}.Restore()
}

func (r Report) Anonymized() Report {
	r.reporter = AccountID{}
	return r
}

func AllAnonymized(reports []Report) []Report {
	anonymized := make([]Report, 0, len(reports))
	for _, report := range reports {
		anonymized = append(anonymized, report.Anonymized())
	}
	return anonymized
}

func (r Report) TakeSnapshot() Snapshot {
	return Snapshot{ID: r.id, Reporter: r.reporter, Tile: r.tile, Complaint: r.complaint, At: r.at}
}

// The state of a Report, for a store. It holds no rule: Restore does.
type Snapshot struct {
	ID ReportID
	// Zero once anonymized.
	Reporter  AccountID
	Tile      TileID
	Complaint Complaint
	At        time.Time
}

func (s Snapshot) Restore() (Report, error) {
	switch {
	case s.ID == ReportID{}:
		return Report{}, fmt.Errorf("%w: no id", ErrInvalidReport)
	case s.Tile == 0:
		return Report{}, fmt.Errorf("%w: no tile", ErrInvalidReport)
	case s.At.IsZero():
		return Report{}, fmt.Errorf("%w: no time", ErrInvalidReport)
	}

	complaint, err := s.Complaint.checked()
	if err != nil {
		return Report{}, err
	}

	return Report{id: s.ID, reporter: s.Reporter, tile: s.Tile, complaint: complaint, at: s.At}, nil
}

type IDProvider interface {
	NewID() (ReportID, error)
}

// A storage finds reports and saves them. It holds no rule: what changes a report is a method of Report.
type Storage interface {
	// A report saved again replaces itself. All of them, or none.
	Save(ctx context.Context, reports ...Report) error
	// What an account filed, oldest first.
	FiledBy(ctx context.Context, reporter AccountID) ([]Report, error)
	// The reports filed before the cutoff that still have a reporter, oldest first, limit at most.
	WithReporterBefore(ctx context.Context, cutoff time.Time, limit int) ([]Report, error)
}
